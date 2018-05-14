package lenses

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kataras/golog"
)

// User represents the logged user, it contains the name, e-mail and the given roles.
type User struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Email string   `json:"email"`
	Roles []string `json:"roles"`
}

// Client is the lenses http client.
// It contains the necessary API calls to communicate and develop via lenses.
type Client struct {
	config Configuration

	// user is generated on `lenses#OpenConnection` when configuration's token is missing,
	// in the same api point that token is generated.
	user User

	// the client is created on the `lenses#OpenConnection` function, it can be customized via options.
	client *http.Client
}

var noOpBuffer = new(bytes.Buffer)

func acquireBuffer(b []byte) *bytes.Buffer {
	if len(b) > 0 {
		// TODO: replace .NewBuffer with pool later on for better performance.
		return bytes.NewBuffer(b)
	}

	return noOpBuffer
}

// isAuthorized is called inside the `Client#do` and it closes the body reader if no accessible.
// 401	Unauthorized	[RFC7235, Section 3.1]
func isAuthorized(resp *http.Response) bool { return resp.StatusCode != http.StatusUnauthorized }

// isOK is called inside the `Client#do` and it closes the body reader if no accessible.
func isOK(resp *http.Response) bool {
	return resp.StatusCode == http.StatusOK ||
		resp.StatusCode == http.StatusCreated || /* see CreateOrUpdateConnector for the `StatusCreated` */
		resp.StatusCode == http.StatusAccepted || /* see PauseConnector for the `StatusAccepted` */
		(resp.Request.Method == http.MethodDelete && resp.StatusCode == http.StatusNoContent) || /* see RemoveConnector for the `StatusNoContnet` */
		(resp.Request.Method == http.MethodPost && resp.StatusCode == http.StatusNoContent) || /* see Restart tasks for the `StatusNoContnet` */
		(resp.StatusCode == http.StatusBadRequest && resp.Request.Method == http.MethodGet) || /* for things like LSQL which can return 400 if invalid query, we need to read the json and print the error message */
		(resp.Request.Method == http.MethodDelete && (resp.StatusCode == http.StatusForbidden) || (resp.StatusCode == http.StatusBadRequest)) /* for things like deletion if not proper user access or invalid value of something passed */
}

const (
	contentTypeHeaderKey = "Content-Type"
	contentTypeJSON      = "application/json"

	xKafkaLensesTokenHeaderKey = "X-Kafka-Lenses-Token"

	acceptHeaderKey          = "Accept"
	acceptEncodingHeaderKey  = "Accept-Encoding"
	contentEncodingHeaderKey = "Content-Encoding"
	gzipEncodingHeaderValue  = "gzip"
)

// ErrCredentialsMissing fires on login, when credentials are missing or
// are invalid or the specific user has no access to a specific action.
var ErrCredentialsMissing = fmt.Errorf("client: credentials missing or invalid")

type requestOption func(r *http.Request)

var schemaAPIOption = func(r *http.Request) {
	r.Header.Add(acceptHeaderKey, contentTypeSchemaJSON)
}

var (
	// ErrResourceNotFound is being fired from all API calls when a 404 not found error code is received.
	// It's a static error message of just `404`, therefore it can be used to add additional info messages based on the caller's action.
	//
	// Example of usage: on topic deletion, if topic does not exist.
	ErrResourceNotFound = fmt.Errorf("%d", http.StatusNotFound)

	// ErrResourceNotAccessible is being fired from all API calls when 403 forbidden code is received.
	// It's static error message, same usage as `ErrResourceNotFound`.
	//
	// Example of usage: on topic's records deletion with offset, if user has no admin access.
	ErrResourceNotAccessible = fmt.Errorf("%d", http.StatusForbidden)

	// ErrResourceNotGood is being fired from all API calls when 400 bad request code is received.
	// It's static error message, same usage as `ErrResourceNotFound`.
	//
	// Example of usage: on topic's records deletion with offset, if offset is negative value.
	ErrResourceNotGood = fmt.Errorf("%d", http.StatusBadRequest)
)

func (c *Client) do(method, path, contentType string, send []byte, options ...requestOption) (*http.Response, error) {
	if path[0] == '/' { // remove beginning slash, if any.
		path = path[1:]
	}

	uri := c.config.Host + "/" + path

	golog.Debugf("Client#do.req:\n\turi: %s:%s\n\tsend: %s", method, uri, string(send))

	req, err := http.NewRequest(method, uri, acquireBuffer(send))
	if err != nil {
		return nil, err
	}
	// before sending requests here.

	// set the token header.
	if c.config.Token != "" {
		req.Header.Set(xKafkaLensesTokenHeaderKey, c.config.Token)
	}

	// set the content type if any.
	if contentType != "" {
		req.Header.Set(contentTypeHeaderKey, contentType)
	}

	// response accept gziped content.
	req.Header.Add(acceptEncodingHeaderKey, gzipEncodingHeaderValue)

	for _, opt := range options {
		opt(req)
	}

	// here will print all the headers, including the token (because it may be useful for debugging)
	// --so bug reporters should be careful here to invalidate the token after that.
	golog.Debugf("Client#do.req.Headers: %#+v", req.Header)

	// send the request and check the response for any connection & authorization errors here.
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	if !isAuthorized(resp) {
		resp.Body.Close() // close the body here so we don't have leaks.
		return nil, ErrCredentialsMissing
	}

	if !isOK(resp) {
		unescapedURI, _ := url.QueryUnescape(uri)
		var errBody string

		if resp.StatusCode != http.StatusNotFound {
			// if the status is not just a 404, then give the whole
			// body to the error context.
			b, err := c.readResponseBody(resp)
			if err != nil {
				errBody = "unable to read body: " + err.Error()
			}

			errBody = "\n" + string(b)
		} else {
			defer resp.Body.Close()
		}

		// if not on debug, then throw static errors, otherwise debug error with the full information, including
		// method, url, status code and the errored body.
		if !c.config.Debug {
			if resp.StatusCode == http.StatusNotFound {
				// if status code is 404, then we set a static error instead of a full message, so front-ends can check and so on.
				return nil, ErrResourceNotFound
			} else if resp.StatusCode == http.StatusForbidden {
				return nil, ErrResourceNotAccessible
			} else if resp.StatusCode == http.StatusBadRequest {
				return nil, ErrResourceNotGood
			}
		}

		return nil, fmt.Errorf("client: (%s: %s) failed with status code %d%s",
			method, unescapedURI, resp.StatusCode, errBody)
	}

	return resp, nil
}

type gzipReadCloser struct {
	respReader io.ReadCloser
	gzipReader io.ReadCloser
}

func (rc *gzipReadCloser) Close() error {
	if rc.gzipReader != nil {
		defer rc.gzipReader.Close()
	}

	return rc.respReader.Close()
}

func (rc *gzipReadCloser) Read(p []byte) (n int, err error) {
	if rc.gzipReader != nil {
		return rc.gzipReader.Read(p)
	}

	return rc.respReader.Read(p)
}

func (c *Client) acquireResponseBodyStream(resp *http.Response) (io.ReadCloser, error) {
	// check for gzip and read it, the right way.
	var (
		reader = resp.Body
		err    error
	)

	if encoding := resp.Header.Get(contentEncodingHeaderKey); encoding == gzipEncodingHeaderValue {
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("client: failed to read gzip compressed content, trace: %v", err)
		}
		// we wrap the gzipReader and the underline response reader
		// so a call of .Close() can close both of them with the correct order when finish reading, the caller decides.
		// Must close manually using a defer on the callers before the `readResponseBody` call,
		// note that the `readJSON` can decide correctly by itself.
		return &gzipReadCloser{
			respReader: resp.Body,
			gzipReader: reader,
		}, nil
	}

	// return the stream reader.
	return reader, err
}

func (c *Client) readResponseBody(resp *http.Response) ([]byte, error) {
	reader, err := c.acquireResponseBodyStream(resp)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(reader)
	if err = reader.Close(); err != nil {
		return nil, err
	}

	if c.config.Debug {
		rawBodyString := string(body)
		// print both body and error, because both of them may be formated by the `readResponseBody`'s caller.
		golog.Debugf("Client#do.resp:\n\tbody: %s\n\terror: %v", rawBodyString, err)
	}

	// return the body.
	return body, err
}

func (c *Client) readJSON(resp *http.Response, valuePtr interface{}) error {
	b, err := c.readResponseBody(resp)
	if err != nil {
		return err
	}

	return json.Unmarshal(b, valuePtr)
}

// GetAccessToken returns the access token that
// generated from the `OpenConnection` or given by the configuration.
func (c *Client) GetAccessToken() string {
	return c.config.Token
}

// User returns the User information from `/api/login`
// received by `OpenConnection`.
func (c *Client) User() User {
	return c.user
} /* we don't expose the token value, unless otherwise requested*/

const logoutPath = "api/logout?token="

// Logout invalidates the token and revoke its access.
// A new Client, using `OpenConnection`, should be created in order to continue after this call.
func (c *Client) Logout() error {
	if c.config.Token == "" {
		return ErrCredentialsMissing
	}

	path := logoutPath + c.config.Token
	resp, err := c.do(http.MethodGet, path, "", nil)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

// LicenseInfo describes the data received from the `GetLicenseInfo`.
type LicenseInfo struct {
	ClientID    string `json:"clientId"`
	IsRespected bool   `json:"isRespected"`
	MaxBrokers  int    `json:"maxBrokers"`
	MaxMessages int    `json:"maxMessages,omitempty"`
	Expiry      int64  `json:"expiry"`

	// no-payload data.

	// ExpiresAt is the time.Time expiration datetime (unix).
	ExpiresAt time.Time `json:"-"`

	// ExpiresDur is the duration that expires from now.
	ExpiresDur time.Duration `json:"-"`

	// YearsToExpire is the length of years that expires from now.
	YearsToExpire int `json:"yearsToExpire,omitempty"`
	// MonthsToExpire is the length of months that expires from now.
	MonthsToExpire int `json:"monthsToExpire,omitempty"`
	// DaysToExpire is the length of days that expires from now.
	DaysToExpire int `json:"daysToExpire,omitempty"`
}

const licensePath = "/api/license"

// GetLicenseInfo returns the license information for the connected lenses box.
func (c *Client) GetLicenseInfo() (LicenseInfo, error) {
	var lc LicenseInfo

	resp, err := c.do(http.MethodGet, licensePath, "", nil)
	if err != nil {
		return lc, err
	}

	if err = c.readJSON(resp, &lc); err != nil {
		return lc, err
	}

	lc.ExpiresAt = time.Unix(lc.Expiry/1000, 0)
	lc.ExpiresDur = lc.ExpiresAt.Sub(time.Now())
	lc.DaysToExpire = int(lc.ExpiresDur.Hours() / 24)
	lc.MonthsToExpire = int(lc.DaysToExpire / 30)
	lc.YearsToExpire = int(lc.MonthsToExpire / 12)

	if lc.YearsToExpire > 0 {
		lc.DaysToExpire = 0
		lc.MonthsToExpire = 0
	} else if lc.MonthsToExpire > 0 {
		lc.DaysToExpire = 0
	}

	return lc, nil
}

// ExecutionMode is the type for the config's execution modes,
// valid values are: IN_PROC/CONNECT/KUBERNETES.
type ExecutionMode string

const (
	// ExecutionModeInvalid represents no mode, this is here for invalid executions mode that
	// maybe returned from the server, maybe useful for the future.
	ExecutionModeInvalid ExecutionMode = "INVALID"
	// ExecutionModeInProcess represents the execution mode IN_PROC.
	ExecutionModeInProcess ExecutionMode = "IN_PROC"
	// ExecutionModeConnect represents the execution mode CONNECT.
	ExecutionModeConnect ExecutionMode = "CONNECT"
	// ExecutionModeKubernetes represents the execution mode KUBERNETES.
	ExecutionModeKubernetes ExecutionMode = "KUBERNETES"
)

// MatchExecutionMode returns the mode based on the string represetantion of it
// and a boolean if that mode is exist or not, the mode will always return in uppercase,
// the input argument is not case sensitive.
//
// The value is just a string but we do this to protect users from mistakes
// or future releases maybe remove/change or replace a string will be much easier.
func MatchExecutionMode(modeStr string) (ExecutionMode, bool) {
	modeStr = strings.ToUpper(modeStr)
	switch modeStr {
	case "IN_PROC":
		return ExecutionModeInProcess, true
	case "CONNECT":
		return ExecutionModeConnect, true
	case "KUBERNETES":
		return ExecutionModeKubernetes, true
	default:
		return ExecutionModeInvalid, false
	}
}

const (
	configPath = "/api/config"
)

// GetConfig returns the whole configuration of the lenses box,
// which can be changed from box to box and it's read-only,
// therefore it returns a map[string]interface{} based on the
// json response body.
//
// To retrieve the execution mode of the box with safety,
// see the `Client#GetExecutionMode` instead.
func (c *Client) GetConfig() (map[string]interface{}, error) {
	resp, err := c.do(http.MethodGet, configPath, "", nil, func(r *http.Request) {
		r.Header.Set("Accept", "application/json, text/plain")
	})

	if err != nil {
		return nil, err
	}

	data := make(map[string]interface{}, 0) // maybe make those statically as well, we'll see.
	if err = c.readJSON(resp, &data); err != nil {
		return nil, err
	}
	return data, nil
}

// GetConfigEntry reads the lenses back-end configuration and sets the value of a key, based on "keys", to the "outPtr".
func (c *Client) GetConfigEntry(outPtr interface{}, keys ...string) error {
	config, err := c.GetConfig()
	if err != nil || config == nil {
		return fmt.Errorf("%s: cannot be extracted: unable to retrieve the config: %v", keys, err)
	}

	// support many tries.
	for i, key := range keys {
		raw, ok := config[key]
		if !ok { // check for existence.
			if isLast := len(keys)-1 == i; !isLast {
				continue
			}
			return fmt.Errorf("%s: couldn't find the corresponding key from config", key)
		}

		// config key found, now exit on the first failure.

		if strPtr, ok := outPtr.(*string); ok {
			// safe cast to string.
			strValue, ok := raw.(string)
			if !ok {
				return fmt.Errorf("%s: %v not type of string", key, raw)
			}

			if len(strValue) == 0 {
				return nil
			}

			// if the outPtr is a raw string, then set it as it's.
			*strPtr = strValue
			return nil
		}

		if intPtr, ok := outPtr.(*int); ok {
			// safe cast to int.
			intValue, ok := raw.(int)
			if !ok {
				return fmt.Errorf("%s: %v not type of int", key, raw)
			}
			// if the outPtr is a raw int, then set it as int.
			*intPtr = intValue
			return err
		}

		// otherwise convert that raw interface to bytes, unmarshal it and set it.
		b, err := json.Marshal(raw)
		if err != nil {
			return err
		}

		// if empty object or empty list or empty quoted string.
		if len(b) <= 2 {
			// fixes
			// lenses.sql.connect.clusters: json unarshal of: '""': json: cannot unmarshal string into Go value of type
			// []lenses.ConnectCluster
			return nil
		}
		if err = json.Unmarshal(b, outPtr); err != nil {
			return fmt.Errorf("%s: json unarshal of: '%s': %v", key, string(b), err)
		}

		return nil
	}

	return nil
}

const executionModeKey = "lenses.sql.execution.mode"

// GetExecutionMode returns the execution mode, if not error returned
// then the possible values are: ExecutionModeInProc, ExecutionModeConnect or ExecutionModeKubernetes.
func (c *Client) GetExecutionMode() (ExecutionMode, error) {
	var modeStr string
	if err := c.GetConfigEntry(&modeStr, executionModeKey); err != nil {
		return ExecutionModeInvalid, err
	}
	return ExecutionMode(modeStr), nil
}

// ConnectCluster contains the connect cluster information that is returned by the `GetConnectClusters` call.
type ConnectCluster struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Statuses string `json:"statuses"`
	Config   string `json:"config"`
	Offsets  string `json:"offsets"`
}

const connectClustersKey = "lenses.connect.clusters"

// GetConnectClusters returns the `lenses.connect.clusters` key from the lenses configuration (`GetConfig`).
func (c *Client) GetConnectClusters() (clusters []ConnectCluster, err error) {
	err = c.GetConfigEntry(&clusters, connectClustersKey)
	return
}

// LSQL API

// LSQLValidation contains the necessary information about an invalid lenses query, see `ValidateLSQL`.
// Example Error:
// {
//     "IsValid": false,
//     "Line": 4,
//     "Column": 1,
//     "Message": "Invalid syntax.Encountered \"LIIT\" at line 4, column 1.\nWas expecting one of:\n    <EOF> ... "
// }
type LSQLValidation struct {
	IsValid bool   `json:"isValid"`
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Message string `json:"message"`
}

var errSQLEmpty = fmt.Errorf("client: sql is empty")

const validateLSQLPath = "api/sql/validation?sql="

// ValidateLSQL validates but not executes a specific LSQL.
func (c *Client) ValidateLSQL(sql string) (v LSQLValidation, err error) {
	if sql == "" {
		err = errSQLEmpty
		return
	}

	path := validateLSQLPath + url.QueryEscape(sql)
	resp, respErr := c.do(http.MethodGet, path, contentTypeJSON, nil)
	if respErr != nil {
		err = respErr
		return
	}

	err = c.readJSON(resp, &v)
	return
}

type (
	// LSQLRecord and LSQLStop and LSQLOffset and LSQLError and optional LSQLStats are the structures that various LSQL information
	// are stored by the SSE client-side, see `LSQL` for more.
	LSQLRecord struct {
		Timestamp int64  `json:"timestamp"`
		Partition int    `json:"partition"`
		Key       string `json:"key"`
		Offset    int    `json:"offset"`
		Topic     string `json:"topic"`
		Value     string `json:"value"` // represents a json object, in raw string.
	}

	// LSQLStop the form of the stop record data that LSQL call returns once.
	LSQLStop struct {
		// If false `max.time` was reached.
		IsTimeRemaining bool `json:"isTimeRemaining"`
		// If true there was no more data on the topic and `max.zero.polls` was reached.
		IsTopicEnd bool `json:"isTopicEnd"`
		// If true the query has been stopped by admin  (Cancel query equivalence).
		IsStopped bool `json:"isStopped"`
		// Number of records read from Kafka.
		TotalRecords int `json:"totalRecords"`
		// Number of records not matching the filter.
		SkippedRecords int `json:"skippedRecords"`
		// Max number of records to pull (driven by LIMIT X,
		// if LIMIT is not present it gets the default config in LENSES).
		RecordsLimit int `json:"recordsLimit"`
		// Total size in bytes read from Kafka.
		TotalSizeRead int64 `json:"totalSizeRead"`
		// Total size in bytes (Kafka size) for the records.
		Size int64 `json:"size"`
		// The topic offsets.
		// If query parameter `&offsets=true` is not present it won't pull the details.
		Offsets []LSQLOffset `json:"offsets"`
	}

	// LSQLOffset the form of the offset record data that LSQL call returns once.
	LSQLOffset struct {
		Partition int   `json:"partition"`
		Min       int64 `json:"min"`
		Max       int64 `json:"max"`
	}

	// LSQLError the form of the error record data that LSQL call returns once.
	LSQLError struct {
		FromLine   int    `json:"fromLine"`
		ToLine     int    `json:"toLine"`
		FromColumn int    `json:"fromColumn"`
		ToColumn   int    `json:"toColumn"`
		Message    string `json:"error"`
	}

	// LSQLStats the form of the stats record data that LSQL call returns.
	LSQLStats struct {
		// Number of records read from Kafka so far.
		TotalRecords int `json:"totalRecords"`
		// Number of records not matching the filter.
		RecordsSkipped int `json:"recordsSkipped"`
		// Max number of records to pull (driven by LIMIT X,
		// if LIMIT is not present it gets the default config in LENSES).
		RecordsLimit int `json:"recordsLimit"`
		// Data read so far in bytes.
		TotalBytes int64 `json:"totalBytes"`
		// Max data allowed in bytes  (driven by `max.bytes`= X,
		// if is not present it gets the default config in LENSES).
		MaxSize int64 `json:"maxSize"`
		// CurrentSize represents the data length accepted so far in bytes (these are records passing the filter).
		CurrentSize int64 `json:"currentSize"`
	}

	// LSQLRecordHandler and LSQLStopHandler and LSQLStopErrorHandler and optionally LSQLStatsHandler
	// describe type of functions that accepts LSQLRecord, LSQLStop, LSQLError and LSQLStats respectfully, and return an error if error not nil then client stops reading from SSE.
	// It's used by the `LSQL` function.
	LSQLRecordHandler func(LSQLRecord) error
	// LSQLStopHandler describes the form of the function that should be registered to accept stop record data from the LSQL call once.
	LSQLStopHandler func(LSQLStop) error
	// LSQLStopErrorHandler describes the form of the function that should be registered to accept error record data from the LSQL call once.
	LSQLStopErrorHandler func(LSQLError) error
	// LSQLStatsHandler describes the form of the function that should be registered to accept stats record data from the LSQL call.
	LSQLStatsHandler func(LSQLStats) error
)

func (err LSQLError) Error() string {
	return err.Message
}

const lsqlPath = "api/sql/data?sql="

var dataPrefix = []byte("data")

// 0- heartbeat. payload is just: "0"
// 1- this represents a record (previousely it was 0).
// 2- represents the end record (previousely it was 1)
// 3- represents an error . it also represents the end (previousely it was 2)
// 4- represents the stats record (previousely it was 3)
var shiftN = len(dataPrefix) + 1 // data:0message, i.e [len(dataPrefix)] == ':' so +1 == '0'.

const (
	heartBeatPayloadType = '0'
	recordPayloadType    = '1'
	stopPayloadType      = '2'
	errPayloadType       = '3'
	statsPayloadType     = '4'
)

// LSQL runs a lenses query and fires the necessary handlers given by the caller.
// Example:
// err := client.LSQL("SELECT * FROM reddit_posts LIMIT 50", true, 2 * time.Second, recordHandler, stopHandler, stopErrHandler, statsHandler)
func (c *Client) LSQL(
	sql string, withOffsets bool, statsEvery time.Duration,
	recordHandler LSQLRecordHandler,
	stopHandler LSQLStopHandler,
	stopErrHandler LSQLStopErrorHandler,
	statsHandler LSQLStatsHandler) error {

	if sql == "" {
		return errSQLEmpty
	}

	withStop := stopHandler != nil

	statsEverySeconds := int(statsEvery.Seconds())
	withStats := statsHandler != nil && statsEverySeconds > 1

	path := lsqlPath + url.QueryEscape(sql)
	// no need to use the url package for these, remember: we have already the ? on the `lsqlPath`.
	if withOffsets {
		path += "&offsets=true"
	}

	if withStats {
		path += fmt.Sprintf("&stats=%d", statsEverySeconds)
	}

	// it's sse, so accept text/event-stream and stream reading the response body, no
	// external libraries needed, it is fairly simple.
	resp, err := c.do(http.MethodGet, path, contentTypeJSON, nil, func(r *http.Request) {
		r.Header.Add(acceptHeaderKey, "application/json, text/event-stream")
	}, schemaAPIOption)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	reader, err := c.acquireResponseBodyStream(resp)
	if err != nil {
		return err
	}

	streamReader := bufio.NewReader(reader)

	for {
		line, err := streamReader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return nil // we read until the the end, exit with no error here.
			}
			return err // exit on first failure.
		}

		if len(line) < shiftN+1 { // even more +1 for the actual event.
			// almost empty or totally invalid line,
			// empty message maybe,
			// we don't care, we ignore them at any way.
			continue
		}

		if !bytes.HasPrefix(line, dataPrefix) {
			return fmt.Errorf("client: see: fail to read the event, the incoming message has no %s prefix", string(dataPrefix))
		}

		messageType := line[shiftN] // we need the [0] here.
		message := line[shiftN+1:]  // we need everything after the '0' here , so shiftN+1.

		switch messageType {
		case heartBeatPayloadType:
			break // ignore.
		case recordPayloadType:
			record := LSQLRecord{}
			if err = json.Unmarshal(message, &record); err != nil {
				// exit on first error here as well.
				return err
			}

			if err = recordHandler(record); err != nil {
				return err
			}
			break
		case stopPayloadType:
			if !withStop {
				return nil
			}

			stopMessage := LSQLStop{}
			if err = json.Unmarshal(message, &stopMessage); err != nil {
				return err
			}

			// And STOP.
			return stopHandler(stopMessage)
		case errPayloadType:
			errMessage := LSQLError{}
			if err = json.Unmarshal(message, &errMessage); err != nil {
				return err
			}

			// And STOP.
			return stopErrHandler(errMessage)
		case statsPayloadType:
			// here we don't necessarily need to check
			// for the existence of the stats handler
			// because the server sends those stats only
			// if the query param exists and it exists if the stats handler != nil,
			// and we did that check already.
			// BUT we make that check here too to prevent any future panics if back-end change.
			if !withStats {
				break
			}

			statsMessage := LSQLStats{}
			if err = json.Unmarshal(message, &statsMessage); err != nil {
				return err
			}

			if err = statsHandler(statsMessage); err != nil {
				return err
			}

			break
		default:
			return fmt.Errorf("client: sse: unknown event received: %s", string(line))
		}
	}
}

// LSQLWait same as `LSQL` but waits until stop or error to return the query's results records, the stats and the stop information.
func (c *Client) LSQLWait(sql string, withOffsets bool, statsEvery time.Duration) (records []LSQLRecord, stats LSQLStats, stop LSQLStop, err error) {
	c.LSQL(sql, withOffsets, statsEvery,
		func(r LSQLRecord) error {
			records = append(records, r)
			return nil
		},
		func(s LSQLStop) error {
			stop = s
			return nil
		},
		func(e LSQLError) error {
			err = e
			return nil
		},
		func(s LSQLStats) error {
			stats = s
			return nil
		},
	)

	return
}

const queriesPath = "api/sql/queries"

// LSQLRunningQuery is the form of the data that the `GetRunningQueries` returns.
type LSQLRunningQuery struct {
	ID        int64  `json:"id"`
	SQL       string `json:"sql"`
	User      string `json:"user"`
	Timestamp int64  `json:"ts"`
}

// GetRunningQueries returns a list of the current sql running queries.
func (c *Client) GetRunningQueries() ([]LSQLRunningQuery, error) {
	resp, err := c.do(http.MethodGet, queriesPath, "", nil)
	if err != nil {
		return nil, err
	}

	var queries []LSQLRunningQuery
	err = c.readJSON(resp, &queries)
	return queries, err
}

// CancelQuery stops a running query based on its ID.
// It returns true whether it was cancelled otherwise false or/and error.
func (c *Client) CancelQuery(id int64) (bool, error) {
	path := fmt.Sprintf(queriesPath+"/%d", id)
	resp, err := c.do(http.MethodDelete, path, "", nil)
	if err != nil {
		return false, err
	}

	var canceled bool
	err = c.readJSON(resp, &canceled)
	return canceled, err
}

// Topics API
//
// Follow the instructions on http://lenses.stream/dev/lenses-apis/rest-api/index.html#topic-api and read
// the call comments for a deeper understanding.

// KV is just a keyvalue map, a form of map[string]interface{}.
type KV map[string]interface{}

var errRequired = func(field string) error {
	return fmt.Errorf("client: %s is required", field)
}

const topicsPath = "api/topics"

// GetTopics returns the list of topics.
func (c *Client) GetTopics() (topics []Topic, err error) {
	// # List of topics
	// GET /api/topics
	// https://docs.confluent.io/current/kafka-rest/docs/api.html#get--topics (in that doc it says a list of topic names but it returns the full topics).
	resp, respErr := c.do(http.MethodGet, topicsPath, "", nil)
	if respErr != nil {
		err = respErr
		return
	}

	err = c.readJSON(resp, &topics)
	return
}

// GetTopicsNames returns the list of topics' names.
func (c *Client) GetTopicsNames() ([]string, error) {
	topics, err := c.GetTopics()
	if err != nil {
		return nil, err
	}

	topicNames := make([]string, len(topics))
	for i := range topics {
		topicNames[i] = topics[i].TopicName
	}

	return topicNames, nil
}

type (
	// TopicMetadata describes the data received from the `GetTopicsMetadata`
	// and the payload to send on the `CreateTopicMetadata`.
	TopicMetadata struct {
		KeyType     string                   `json:"keyType" yaml:"KeyType"`
		ValueType   string                   `json:"valueType" yaml:"ValueType"`
		TopicName   string                   `json:"topicName" yaml:"TopicName"`
		ValueSchema TopicMetadataValueSchema `json:"valueSchema" yaml:"ValueSchema"`
		KeySchema   TopicMetadataKeySchema   `json:"keySchema" yaml:"KeySchema"`
	}

	// TopicMetadataValueSchema describes the "ValueSchema" field of the `TopicMetadata` structure.
	TopicMetadataValueSchema struct {
		Type      string               `json:"type" yaml:"Type"`
		Name      string               `json:"name" yaml:"Name"`
		Namespace string               `json:"namespace" yaml:"Namespace"`
		Doc       string               `json:"doc" yaml:"Doc"`
		Fields    []TopicMetadataField `json:"fields" yaml:"Fields"`
	}

	// TopicMetadataField contains the "Name" and the "Type" of a topic metadata field.
	//
	// See `TopicMetadataValueSchema` and `TopicMetadataKeySchema` for more.
	TopicMetadataField struct {
		Name string `json:"name" yaml:"Name"`
		Type string `json:"type" yaml:"Type"`
	}

	// TopicMetadataKeySchema describes the "KeySchema" field of the `TopicMetadata` structure.
	TopicMetadataKeySchema struct {
		Type      string               `json:"type" yaml:"Type"`
		Name      string               `json:"name" yaml:"Name"`
		Namespace string               `json:"namespace" yaml:"Namespace"`
		Fields    []TopicMetadataField `json:"fields" yaml:"Fields"`
	}
)

const (
	topicsMetadataPath = "/api/system/topics/metadata"
	topicMetadataPath  = topicsMetadataPath + "/%s"
)

// GetTopicsMetadata retrieves and returns all the topics' available metadata.
func (c *Client) GetTopicsMetadata() ([]TopicMetadata, error) {
	resp, err := c.do(http.MethodGet, topicsMetadataPath, "", nil)
	if err != nil {
		return nil, err
	}

	var meta []TopicMetadata

	err = c.readJSON(resp, &meta)
	return meta, err
}

func (c *Client) GetTopicMetadata(topicName string) (TopicMetadata, error) {
	var meta TopicMetadata

	if topicName == "" {
		return meta, errRequired("topicName")
	}

	path := fmt.Sprintf(topicMetadataPath, topicName)
	resp, err := c.do(http.MethodGet, path, "", nil)
	if err != nil {
		return meta, err
	}

	err = c.readJSON(resp, &meta)
	return meta, err
}

// CreateTopicMetadata adds a topic metadata.
func (c *Client) CreateTopicMetadata(metadata TopicMetadata) error {
	if metadata.TopicName == "" {
		return errRequired("metadata.TopicName")
	}

	send, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	resp, err := c.do(http.MethodPost, topicsMetadataPath, contentTypeJSON, send)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

// DeleteTopicMetadata removes an existing topic metadata.
func (c *Client) DeleteTopicMetadata(topicName string) error {
	if topicName == "" {
		return errRequired("topicName")
	}

	path := fmt.Sprintf(topicMetadataPath, topicName)
	resp, err := c.do(http.MethodDelete, path, "", nil)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

// CreateTopicPayload contains the data that the `CreateTopic` accepts, as a single structure.
type CreateTopicPayload struct {
	TopicName   string `json:"topicName" yaml:"Name"`
	Replication int    `json:"replication" yaml:"Replication"`
	Partitions  int    `json:"partitions" yaml:"Partitions"`
	Configs     KV     `json:"configs" yaml:"Configs"`
}

// CreateTopic creates a topic.
//
// topicName, string, Required.
// replication, int.
// partitions, int.
// configs, topic key - value.
//
// Read more at: http://lenses.stream/dev/lenses-apis/rest-api/index.html#create-topic
func (c *Client) CreateTopic(topicName string, replication, partitions int, configs KV) error {
	if topicName == "" {
		return errRequired("topicName")
	}

	payload := CreateTopicPayload{
		TopicName:   topicName,
		Replication: replication,
		Partitions:  partitions,
		Configs:     configs,
	}

	send, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := c.do(http.MethodPost, topicsPath, contentTypeJSON, send)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

const (
	topicPath        = topicsPath + "/%s"
	topicRecordsPath = topicPath + "/%d/%d"
)

// DeleteTopic deletes a topic.
// It accepts the topicName, a required, not empty string.
//
// Read more at: http://lenses.stream/dev/lenses-apis/rest-api/index.html#delete-topic
func (c *Client) DeleteTopic(topicName string) error {
	if topicName == "" {
		return errRequired("topicName")
	}

	path := fmt.Sprintf(topicPath, topicName)
	resp, err := c.do(http.MethodDelete, path, "", nil)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

// DeleteTopicRecords deletes a topic's records from partition to an offset.
// If user has no rights for that action it returns `ErrResourceNotAccessible`,
// if negative value of "toOffset" then it returns `ErrResourceNotGood`.
//
// All input arguments are required.
func (c *Client) DeleteTopicRecords(topicName string, fromPartition int, toOffset int64) error {
	if topicName == "" {
		return errRequired("topicName")
	}

	if toOffset < 0 || fromPartition < 0 {
		return ErrResourceNotGood
	}

	path := fmt.Sprintf(topicRecordsPath, topicName, fromPartition, toOffset)
	resp, err := c.do(http.MethodDelete, path, "", nil)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

const updateTopicConfigPath = topicsPath + "/config/%s"

// UpdateTopicPayload contains the data that the `CreateTopic` accepts, as a single structure.
type UpdateTopicPayload struct {
	Name    string `json:"name,omitempty" yaml:"Name"` // empty for request send, filled for cli.
	Configs []KV   `json:"configs,omitempty" yaml:"Configs"`
}

// UpdateTopic updates a topic's configuration.
// topicName, string.
// configsSlice, array of topic config key-values.
//
// Read more at: http://lenses.stream/dev/lenses-apis/rest-api/index.html#update-topic-configuration
func (c *Client) UpdateTopic(topicName string, configsSlice []KV) error {
	if topicName == "" {
		return errRequired("topicName")
	}

	payload := UpdateTopicPayload{Configs: configsSlice}

	send, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	path := fmt.Sprintf(updateTopicConfigPath, topicName)
	resp, err := c.do(http.MethodPut, path, contentTypeJSON, send)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

// Topic describes the data that the `CreateTopic` returns.
type Topic struct {
	TopicName            string             `json:"topicName"`
	KeyType              string             `json:"keyType"`   // maybe string-based enum?
	ValueType            string             `json:"valueType"` // maybe string-based enum?
	Partitions           int                `json:"partitions"`
	Replications         int                `json:"replications"`
	IsControlTopic       bool               `json:"isControlTopic"`
	KeySchema            string             `json:"keySchema,omitempty"`
	ValueSchema          string             `json:"valueSchema,omitempty"`
	MessagesPerSecond    int64              `json:"messagesPerSecond"`
	TotalMessages        int64              `json:"totalMessages"`
	Timestamp            int64              `json:"timestamp"`
	IsMarkedForDeletion  bool               `json:"isMarkedForDeletion"`
	Config               []KV               `json:"config"`
	ConsumersGroup       []ConsumersGroup   `json:"consumers"`
	MessagesPerPartition []PartitionMessage `json:"messagesPerPartition"`
}

// ConsumersGroup describes the data that the `Topic`'s  `ConsumersGroup` field contains.
type ConsumersGroup struct {
	ID          string              `json:"id"`
	Coordinator ConsumerCoordinator `json:"coordinator"`
	// On consumers not active/committing offsets - we don't get any of the following info
	Active               bool               `json:"active"`
	State                ConsumerGroupState `json:"state"`
	Consumers            []string           `json:"consumers"`
	ConsumersCount       int                `json:"consumersCount,omitempty"`
	TopicPartitionsCount int                `json:"topicPartitionsCount,omitempty"`
	MinLag               int64              `json:"minLag,omitempty"`
	MaxLag               int64              `json:"maxLag,omitempty"`
}

// ConsumerGroupState describes the valid values of a `ConsumerGroupState`:
// `StateUnknown`,`StateStable`,`StateRebalancing`,`StateDead`,`StateNoActiveMembers`,`StateExistsNot`,`StateCoordinatorNotFound`.
type ConsumerGroupState string

const (
	// StateUnknown is a valid `ConsumerGroupState` value of "Unknown".
	StateUnknown ConsumerGroupState = "Unknown"
	// StateStable is a valid `ConsumerGroupState` value of "Stable".
	StateStable ConsumerGroupState = "Stable"
	// StateRebalancing is a valid `ConsumerGroupState` value of "Rebalancing".
	StateRebalancing ConsumerGroupState = "Rebalancing"
	// StateDead is a valid `ConsumerGroupState` value of "Dead".
	StateDead ConsumerGroupState = "Dead"
	// StateNoActiveMembers is a valid `ConsumerGroupState` value of "NoActiveMembers".
	StateNoActiveMembers ConsumerGroupState = "NoActiveMembers"
	// StateExistsNot is a valid `ConsumerGroupState` value of "ExistsNot".
	StateExistsNot ConsumerGroupState = "ExistsNot"
	// StateCoordinatorNotFound is a valid `ConsumerGroupState` value of "CoordinatorNotFound".
	StateCoordinatorNotFound ConsumerGroupState = "CoordinatorNotFound"
)

// Consumer describes the consumer valid response data.
type Consumer struct {
	Topic                     string `json:"topic"`
	CurrentOffset             int64  `json:"currentOffset"`
	LogEndOffset              int64  `json:"longEndOffset"`
	Lag                       int64  `json:"lag"`
	ConsumerID                string `json:"consumerId"`
	Host                      string `json:"host"`
	ClientID                  string `json:"clientId"`
	MessagesPerSecond         int64  `json:"messagesPerSecond"`
	ProducerMessagesPerSecond int64  `json:"producerMessagesPerSecond"`
}

// ConsumerCoordinator describes the consumer coordinator's valid response data.
type ConsumerCoordinator struct {
	ID   int    `json:"id"`
	Host string `json:"host"`
	Port int    `json:"port"`
	Rack string `json:"rack"`
}

// PartitionMessage describes a partition's message response data.
type PartitionMessage struct {
	Partition int   `json:"partition"`
	Messages  int64 `json:"messages"`
	Begin     int64 `json:"begin"`
	End       int64 `json:"end"`
}

// GetTopic returns a topic's information, a `lenses.Topic` value.
//
// Read more at: http://lenses.stream/dev/lenses-apis/rest-api/index.html#get-topic-information
func (c *Client) GetTopic(topicName string) (topic Topic, err error) {
	if topicName == "" {
		err = errRequired("topicName")
		return
	}

	path := fmt.Sprintf(topicPath, topicName)
	resp, respErr := c.do(http.MethodGet, path, "", nil)
	if respErr != nil {
		err = respErr
		return
	}

	err = c.readJSON(resp, &topic)
	return
}

// Processor API

const processorsPath = "api/streams"

// CreateProcessorPayload holds the data to be sent from `CreateProcessor`.
type CreateProcessorPayload struct {
	Name        string `json:"name" yaml:"Name"` // required
	SQL         string `json:"sql" yaml:"SQL"`   // required
	Runners     int    `json:"runners" yaml:"Runners"`
	ClusterName string `json:"clusterName" yaml:"ClusterName"`
	Namespace   string `json:"namespace" yaml:"Namespace"`
	Pipeline    string `json:"pipeline" yaml:"Pipeline"` // defaults to Name if not set.
}

// CreateProcessor creates a new LSQL processor.
func (c *Client) CreateProcessor(name string, sql string, runners int, clusterName, namespace, pipeline string) error {
	if name == "" {
		return errRequired("name")
	}

	if sql == "" {
		return errRequired("sql")
	}

	if runners <= 0 {
		runners = 1
	}

	if pipeline == "" {
		pipeline = name
	}

	payload := CreateProcessorPayload{
		Name:        name,
		SQL:         sql,
		Runners:     runners,
		ClusterName: clusterName,
		Namespace:   namespace,
		Pipeline:    pipeline,
	}

	send, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := c.do(http.MethodPost, processorsPath, contentTypeJSON, send)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

type (
	// ProcessorsResult describes the data that are being received from the `GetProcessors`.
	ProcessorsResult struct {
		Targets []ProcessorTarget `json:"targets"`
		Streams []ProcessorStream `json:"streams"`
	}

	// ProcessorTarget describes the processor target,
	// see `ProcessorResult`.
	ProcessorTarget struct {
		Cluster    string   `json:"cluster"`
		Version    string   `json:"version,omitempty"`
		Namespaces []string `json:"namespaces"`
	}

	// ProcessorStream describes the processor stream,
	// see `ProcessorResult`.
	ProcessorStream struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		ClusterName string `json:"clusterName"`
		User        string `json:"user"`
		Namespace   string `json:"namespace"`
		Uptime      int64  `json:"uptime"`

		SQL                    string `json:"sql"`
		Runners                int    `json:"runners"`
		DeploymentState        string `json:"deploymentState"`
		TopicValueDecoder      string `json:"topicValueDecoder"`
		Pipeline               string `json:"pipeline"`
		StartTimestamp         int64  `json:"startTs"`
		StopTimestamp          int64  `json:"stopTs,omitempty"`
		ToTopic                string `json:"toTopic"`
		LastActionMessage      string `json:"lastActionMsg,omitempty"`
		DeploymentErrorMessage string `json:"deploymentErrorMsg,omitempty"`

		RunnerState map[string]ProcessorRunnerState `json:"runnerState"`
	}
	// ProcessorRunnerState describes the processor stream,
	// see `ProcessorStream` and `ProcessorResult.
	ProcessorRunnerState struct {
		ID           string `json:"id"`
		Worker       string `json:"worker"`
		State        string `json:"state"`
		ErrorMessage string `json:"errorMsg"`
	}
)

// GetProcessors returns a list of all available LSQL processors.
func (c *Client) GetProcessors() (ProcessorsResult, error) {
	var res ProcessorsResult

	resp, err := c.do(http.MethodGet, processorsPath, "", nil)
	if err != nil {
		return res, err
	}

	if err = c.readJSON(resp, &res); err != nil {
		return res, err
	}

	return res, nil
}

// LookupProcessorIdentifier is not a direct API call, although it fires requests to get the result.
// It's a helper which can be used as an input argument of the `DeleteProcessor` and `PauseProcessor` and `ResumeProcessor` and `UpdateProcessorRunners` functions.
//
// Fill the id or name in any case.
// Fill the clusterName and namespace when in KUBERNETES execution mode.
func (c *Client) LookupProcessorIdentifier(id, name, clusterName, namespace string) (string, error) {
	if name == "" && id == "" {
		return "", fmt.Errorf("LookupProcessorIdentifier: name or id are missing")
	}

	mode, err := c.GetExecutionMode()
	if err != nil {
		return "", err // unable to determinate the lenses execution mode.
	}

	identifier := name

	if mode == ExecutionModeConnect || mode == ExecutionModeInProcess {
		if id != "" {
			identifier = id
		} else if name != "" {
			// get the id by looping over all available processors.
			result, err := c.GetProcessors()
			if err != nil {
				return "", err
			}

			for _, processor := range result.Streams {
				if processor.Name == name {
					// Just an information:
					// if mode is IN_PROC, then the below processor.ID is: the pipeline prefix followed by `_` as well.
					identifier = processor.ID
					break
				}
			}

		} else {
			return "", fmt.Errorf("LookupProcessorIdentifier: name or id arguments are missing")
		}

	} else if mode == ExecutionModeKubernetes {
		if id != "" {
			identifier = id
		} else {
			// the clusterName+.+namespace+.+processor name is the string we need in the endpoints,
			// therefore, we require both cluster name and namespace in K8.
			if clusterName == "" || namespace == "" || name == "" {
				return "", fmt.Errorf("LookupProcessorIdentifier:KUBERNETES: (name or clusterName or namespace) or id arguments are missing")
			}

			identifier = fmt.Sprintf("%s.%s.%s", clusterName, namespace, name)
		}
	}

	return identifier, nil
}

const processorPath = processorsPath + "/%s"

const processorPausePath = processorPath + "/pause"

// PauseProcessor pauses a processor.
// See `LookupProcessorIdentifier`.
func (c *Client) PauseProcessor(processorID string) error {
	if processorID == "" {
		return errRequired("processorID")
	}

	path := fmt.Sprintf(processorPath+"/pause", processorID)
	resp, err := c.do(http.MethodPut, path, "", nil)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

const processorResumePath = processorPath + "/resume"

// ResumeProcessor resumes a processor.
// See `LookupProcessorIdentifier`.
func (c *Client) ResumeProcessor(processorID string) error {
	if processorID == "" {
		return errRequired("processorID")
	}

	path := fmt.Sprintf(processorResumePath, processorID)
	resp, err := c.do(http.MethodPut, path, "", nil)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

const processorUpdateRunnersPath = processorPath + "/scale/%d"

// UpdateProcessorRunners scales a processor to "numberOfRunners".
// See `LookupProcessorIdentifier`.
func (c *Client) UpdateProcessorRunners(processorID string, numberOfRunners int) error {
	if processorID == "" {
		return errRequired("processorID")
	}

	if numberOfRunners <= 0 {
		numberOfRunners = 1
	}

	path := fmt.Sprintf(processorUpdateRunnersPath, processorID, numberOfRunners)
	resp, err := c.do(http.MethodPut, path, "", nil)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

// DeleteProcessor removes a processor based on its name or the full id,
// it depends on lenses execution mode, use the `LookupProcessorIdentifier`.
func (c *Client) DeleteProcessor(processorNameOrID string) error {
	if processorNameOrID == "" {
		return errRequired("processorNameOrID")
	}

	path := fmt.Sprintf(processorPath, processorNameOrID)
	resp, err := c.do(http.MethodDelete, path, "", nil)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

// Connector API
//
// https://docs.confluent.io/current/connect/devguide.html
// https://docs.confluent.io/current/connect/restapi.html
// http://lenses.stream/dev/lenses-apis/rest-api/index.html#connector-api

// ConnectorConfig the configuration parameters
// for the connector.
//
// For both send and receive:
// https://docs.confluent.io/current/connect/restapi.html#post--connectors
type ConnectorConfig map[string]interface{}

// ConnectorTaskReadOnly is the type that returned
// as "tasks" from the connector, it's for read-only access,
// it contains the basic information about the connector's task.
// It usually returned as a slice of ConnectorTaskReadOnly.
//
// See `Connector` type for more.
type ConnectorTaskReadOnly struct {
	// Connector is the name of the connector the task belongs to.
	Connector string `json:"connector"`
	// Task is the Task ID within the connector.
	Task int `json:"task"`
}

// Connector contains the connector's information, both send and receive.
type Connector struct {
	// https://docs.confluent.io/current/connect/restapi.html#get--connectors-(string-name)

	ClusterName string `json:"clusterName,omitempty"` // internal use only, not set by response.
	// Name of the created (or received) connector.
	Name string `json:"name"`
	// Config parameters for the connector
	Config ConnectorConfig `json:"config,omitempty"`
	// Tasks is the list of active tasks generated by the connector.
	Tasks []ConnectorTaskReadOnly `json:"tasks,omitempty"`
}

const connectorsPath = "/api/proxy-connect/%s/connectors"

// GetConnectors returns a list of active connectors names as list of strings.
//
// Visit http://lenses.stream/dev/lenses-apis/rest-api/index.html#connector-api
// and https://docs.confluent.io/current/connect/restapi.html for a deeper understanding.
func (c *Client) GetConnectors(clusterName string) (names []string, err error) {
	if clusterName == "" {
		err = errRequired("clusterName")
		return
	}

	// # List active connectors
	// GET /api/proxy-connect/(string: clusterName)/connectors
	path := fmt.Sprintf(connectorsPath, clusterName)
	resp, respErr := c.do(http.MethodGet, path, contentTypeJSON, nil)
	if respErr != nil {
		err = respErr
		return
	}

	err = c.readJSON(resp, &names)
	return
}

// CreateUpdateConnectorPayload can be used to hold the data for creating or updating a connector.
type CreateUpdateConnectorPayload struct {
	ClusterName string          `yaml:"ClusterName"`
	Name        string          `yaml:"Name"`
	Config      ConnectorConfig `yaml:"Config"`
}

// ApplyAndValidateName applies some rules to make sure that the connector's data are setup correctly.
func (c *CreateUpdateConnectorPayload) ApplyAndValidateName() error {
	if c.Config != nil {
		value, found := c.Config["name"]
		if found {
			configName, ok := value.(string)
			if !ok {
				return fmt.Errorf(`config["name"] is not type of string`)
			}

			if c.Name != "" && configName != c.Name {
				return fmt.Errorf(`config["name"] '%s' and name '%s' do not match`, configName, c.Name)
			}

			if c.Name == "" {
				c.Name = configName
			}

			return nil
		}

		// if not found in config, then set it from connector.Name if it's there, otherwise fire name missing error.
		if c.Name == "" {
			return fmt.Errorf("name is required")
		}

		c.Config["name"] = c.Name
	}

	return nil
}

// CreateConnector creates a new connector.
// It returns the current connector info if successful.
//
//
// name (string) – Name of the connector to create
// config (map) – Configuration parameters for the connector. All values should be strings.
//
// Read more at: https://docs.confluent.io/current/connect/restapi.html#post--connectors
//
// Look `UpdateConnector` too.
func (c *Client) CreateConnector(clusterName, name string, config ConnectorConfig) (connector Connector, err error) {
	if clusterName == "" {
		err = errRequired("clusterName")
		return
	}

	if name == "" {
		err = errRequired("name")
		return
	}

	connector.Name = name
	connector.Config = config
	send, derr := json.Marshal(connector)
	if derr != nil {
		err = derr
		return
	}

	// # Create new connector
	// POST /api/proxy-connect/(string: clusterName)/connectors [CONNECTOR_CONFIG]
	path := fmt.Sprintf(connectorsPath, clusterName)
	resp, respErr := c.do(http.MethodPost, path, contentTypeJSON, send)
	if respErr != nil {
		err = respErr
		return
	}

	// re-use of the connector payload.
	err = c.readJSON(resp, &connector)
	return
}

// UpdateConnector sets the configuration of an existing connector.
//
// It returns information about the connector after the change has been made
// and an indicator if that connector was created or just configuration update.
func (c *Client) UpdateConnector(clusterName, name string, config ConnectorConfig) (connector Connector, err error) {
	if clusterName == "" {
		err = errRequired("clusterName")
		return
	}

	if name == "" {
		err = errRequired("name")
		return
	}

	send, derr := json.Marshal(config)
	if derr != nil {
		err = derr
		return
	}

	// # Set connector config
	// PUT /api/proxy-connect/(string: clusterName)/connectors/(string: name)/config
	path := fmt.Sprintf(connectorPath+"/config", clusterName, name)
	resp, respErr := c.do(http.MethodPut, path, contentTypeJSON, send)
	if respErr != nil {
		err = respErr
		return
	}

	err = c.readJSON(resp, &connector)
	// updated := !(resp.StatusCode == http.StatusCreated)

	return
}

const connectorPath = connectorsPath + "/%s"

// GetConnector returns the information about the connector.
// See `Connector` type and read more at: https://docs.confluent.io/current/connect/restapi.html#get--connectors-(string-name)
func (c *Client) GetConnector(clusterName, name string) (connector Connector, err error) {
	if clusterName == "" {
		err = errRequired("clusterName")
		return
	}

	if name == "" {
		err = errRequired("name")
		return
	}

	// # Get information about a specific connector
	// GET /api/proxy-connect/(string: clusterName)/connectors/(string: name)
	path := fmt.Sprintf(connectorPath, clusterName, name)
	resp, respErr := c.do(http.MethodGet, path, contentTypeJSON, nil)
	if respErr != nil {
		err = respErr
		return
	}

	err = c.readJSON(resp, &connector)
	connector.ClusterName = clusterName
	return
}

// GetConnectorConfig returns the configuration for the connector.
func (c *Client) GetConnectorConfig(clusterName, name string) (cfg ConnectorConfig, err error) {
	if clusterName == "" {
		err = errRequired("clusterName")
		return
	}

	if name == "" {
		err = errRequired("name")
		return
	}

	// # Get connector config
	// GET /api/proxy-connect/(string: clusterName)/connectors/(string: name)/config
	path := fmt.Sprintf(connectorPath, clusterName, name)
	resp, respErr := c.do(http.MethodGet, path, contentTypeJSON, nil)
	if respErr != nil {
		err = respErr
		return
	}

	err = c.readJSON(resp, &cfg)
	return
}

// ConnectorState indicates the connector status task's state and connector's state.
// As defined at: https://docs.confluent.io/current/connect/managing.html#connector-and-task-status
type ConnectorState string

const (
	// UNASSIGNED state indicates that the connector/task has not yet been assigned to a worker.
	UNASSIGNED ConnectorState = "UNASSIGNED"
	// RUNNING state indicates that the connector/task is running.
	RUNNING ConnectorState = "RUNNING"
	// PAUSED state indicates that the connector/task has been administratively paused.
	PAUSED ConnectorState = "PAUSED"
	// FAILED state indicates that the connector/task has failed
	// (usually by raising an exception, which is reported in the status output).
	FAILED ConnectorState = "FAILED"
)

type (
	// ConnectorStatus describes the data that are being received from the `GetConnectorStatus`.
	ConnectorStatus struct {
		// Name is the name of the connector.
		Name      string                        `json:"name"`
		Connector ConnectorStatusConnectorField `json:"connector"`
		Tasks     []ConnectorStatusTask         `json:"tasks,omitempty"`
	}

	// ConnectorStatusConnectorField describes a connector's status,
	// see `ConnectorStatus`.
	ConnectorStatusConnectorField struct {
		State    string `json:"state"`     // i.e RUNNING
		WorkerID string `json:"worker_id"` // i.e fakehost:8083
	}

	// ConnectorStatusTask describes a connector task's status,
	// see `ConnectorStatus`.
	ConnectorStatusTask struct {
		ID       int    `json:"id"`              // i.e 1
		State    string `json:"state"`           // i.e FAILED
		WorkerID string `json:"worker_id"`       // i.e fakehost:8083
		Trace    string `json:"trace,omitempty"` // i.e org.apache.kafka.common.errors.RecordTooLargeException\n
	}
)

// GetConnectorStatus returns the current status of the connector, including whether it is running,
// failed or paused, which worker it is assigned to, error information if it has failed,
// and the state of all its tasks.
func (c *Client) GetConnectorStatus(clusterName, name string) (cs ConnectorStatus, err error) {
	if clusterName == "" {
		err = errRequired("clusterName")
		return
	}

	if name == "" {
		err = errRequired("name")
		return
	}

	// # Get connector status
	// GET /api/proxy-connect/(string: clusterName)/connectors/(string: name)/status
	path := fmt.Sprintf(connectorPath+"/status", clusterName, name)
	resp, respErr := c.do(http.MethodGet, path, "", nil)
	if respErr != nil {
		err = respErr
		return
	}

	err = c.readJSON(resp, &cs)
	return
}

// PauseConnector pauses the connector and its tasks, which stops message processing until the connector is resumed.
// This call asynchronous and the tasks will not transition to PAUSED state at the same time.
func (c *Client) PauseConnector(clusterName, name string) error {
	if clusterName == "" {
		return errRequired("clusterName")
	}

	if name == "" {
		return errRequired("name")
	}

	// # Pause a connector
	// PUT /api/proxy-connect/(string: clusterName)/connectors/(string: name)/pause
	path := fmt.Sprintf(connectorPath+"/pause", clusterName, name)
	resp, err := c.do(http.MethodPut, path, "", nil) // the success status is 202 Accepted.
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

// ResumeConnector resumes a paused connector or do nothing if the connector is not paused.
// This call asynchronous and the tasks will not transition to RUNNING state at the same time.
func (c *Client) ResumeConnector(clusterName, name string) error {
	if clusterName == "" {
		return errRequired("clusterName")
	}

	if name == "" {
		return errRequired("name")
	}

	// # Resume a paused connector
	// PUT /api/proxy-connect/(string: clusterName)/connectors/(string: name)/resume
	path := fmt.Sprintf(connectorPath+"/resume", clusterName, name)
	resp, err := c.do(http.MethodPut, path, "", nil)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

// RestartConnector restarts the connector and its tasks.
// It returns a 409 (Conflict) status code error if rebalance is in process.
func (c *Client) RestartConnector(clusterName, name string) error {
	if clusterName == "" {
		return errRequired("clusterName")
	}

	if name == "" {
		return errRequired("name")
	}

	// # Restart a connector
	// POST /api/proxy-connect/(string: clusterName)/connectors/(string: name)/restart
	path := fmt.Sprintf(connectorPath+"/restart", clusterName, name)
	resp, err := c.do(http.MethodPost, path, "", nil)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

// DeleteConnector deletes a connector, halting all tasks and deleting its configuration.
// It return a 409 (Conflict) status code error if rebalance is in process.
func (c *Client) DeleteConnector(clusterName, name string) error {
	if clusterName == "" {
		return errRequired("clusterName")
	}

	if name == "" {
		return errRequired("name")
	}

	// # Remove a running connector
	// DELETE /api/proxy-connect/(string: clusterName)/connectors/(string: name)
	path := fmt.Sprintf(connectorPath, clusterName, name)
	resp, err := c.do(http.MethodDelete, path, "", nil)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

const (
	tasksPath = connectorPath + "/tasks"
	taskPath  = tasksPath + "/%d"
)

// GetConnectorTasks returns a list of tasks currently running for the connector.
// Read more at: https://docs.confluent.io/current/connect/restapi.html#get--connectors-(string-name)-tasks.
func (c *Client) GetConnectorTasks(clusterName, name string) (m []map[string]interface{}, err error) {
	if clusterName == "" {
		return nil, errRequired("clusterName")
	}

	if name == "" {
		return nil, errRequired("name")
	}

	// # Get list of connector tasks
	// GET /api/proxy-connect/(string: clusterName)/connectors/(string: name)/tasks
	path := fmt.Sprintf(tasksPath, clusterName, name)
	resp, respErr := c.do(http.MethodGet, path, "", nil)
	if respErr != nil {
		err = respErr
		return
	}

	err = c.readJSON(resp, &m)
	return
}

// GetConnectorTaskStatus returns a task’s status.
func (c *Client) GetConnectorTaskStatus(clusterName, name string, taskID int) (cst ConnectorStatusTask, err error) {
	if clusterName == "" {
		err = errRequired("clusterName")
		return
	}

	if name == "" {
		err = errRequired("name")
		return
	}

	// # Get current status of a task
	// GET /connectors/(string: name)/tasks/(int: taskid)/status in confluent
	path := fmt.Sprintf(taskPath+"/status", clusterName, name, taskID)
	resp, respErr := c.do(http.MethodGet, path, "", nil)
	if respErr != nil {
		err = respErr
		return
	}

	err = c.readJSON(resp, &cst)
	return
}

// RestartConnectorTask restarts an individual task.
func (c *Client) RestartConnectorTask(clusterName, name string, taskID int) error {
	if clusterName == "" {
		return errRequired("clusterName")
	}

	if name == "" {
		return errRequired("name")
	}

	// # Restart a connector task
	// POST /api/proxy-connect/(string: clusterName)/connectors/(string: name)/tasks/(int: taskid)/restart
	path := fmt.Sprintf(taskPath+"/restart", clusterName, name, taskID)
	resp, err := c.do(http.MethodPost, path, "", nil)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

// ConnectorPlugin describes the entry data of the list that are being received from the `GetConnectorPlugins`.
type ConnectorPlugin struct {
	// Class is the connector class name.
	Class string `json:"class"`

	Type string `json:"type"`

	Version string `json:"version"`
}

const pluginsPath = "/api/proxy-connect/%s/connector-plugins"

// GetConnectorPlugins returns a list of connector plugins installed in the Kafka Connect cluster.
// Note that the API only checks for connectors on the worker that handles the request,
// which means it is possible to see inconsistent results,
// especially during a rolling upgrade if you add new connector jars.
func (c *Client) GetConnectorPlugins(clusterName string) (cp []ConnectorPlugin, err error) {
	if clusterName == "" {
		return nil, errRequired("clusterName")
	}

	// # List available connector plugins
	// GET /api/proxy-connect/(string: clusterName)/connector-plugins
	path := fmt.Sprintf(pluginsPath, clusterName)
	resp, respErr := c.do(http.MethodGet, path, "", nil)
	if respErr != nil {
		err = respErr
		return
	}

	err = c.readJSON(resp, &cp)
	return
}

// Schemas (and Subjects) API
// https://docs.confluent.io/current/schema-registry/docs/api.html

const schemaAPIVersion = "v1"
const contentTypeSchemaJSON = "application/vnd.schemaregistry." + schemaAPIVersion + "+json"

const subjectsPath = "api/proxy-sr/subjects"

// GetSubjects returns a list of the available subjects(schemas).
// https://docs.confluent.io/current/schema-registry/docs/api.html#subjects
func (c *Client) GetSubjects() (subjects []string, err error) {
	// # List all available subjects
	// GET /api/proxy-sr/subjects
	resp, respErr := c.do(http.MethodGet, subjectsPath, "", nil, schemaAPIOption)
	if respErr != nil {
		err = respErr
		return
	}

	err = c.readJSON(resp, &subjects)
	return
}

const subjectPath = subjectsPath + "/%s"

// GetSubjectVersions returns all the versions of a subject(schema) based on its name.
func (c *Client) GetSubjectVersions(subject string) (versions []int, err error) {
	if subject == "" {
		err = errRequired("subject")
		return
	}

	// # List all versions of a particular subject
	// GET /api/proxy-sr/subjects/(string: subject)/versions
	path := fmt.Sprintf(subjectPath, subject+"/versions")
	resp, respErr := c.do(http.MethodGet, path, "", nil, schemaAPIOption)
	if respErr != nil {
		err = respErr
		return
	}

	err = c.readJSON(resp, &versions)
	return
}

// DeleteSubject deletes the specified subject and its associated compatibility level if registered.
// It is recommended to use this API only when a topic needs to be recycled or in development environment.
// Returns the versions of the schema deleted under this subject.
func (c *Client) DeleteSubject(subject string) (versions []int, err error) {
	if subject == "" {
		err = errRequired("subject")
		return
	}

	// DELETE /api/proxy-sr/subjects/(string: subject)
	path := fmt.Sprintf(subjectPath, subject)
	resp, respErr := c.do(http.MethodDelete, path, "", nil, schemaAPIOption)
	if respErr != nil {
		err = respErr
		return
	}

	err = c.readJSON(resp, &versions)
	return
}

type schemaOnlyJSON struct {
	Schema string `json:"schema"`
}

const schemaPath = "api/proxy-sr/schemas/ids/%d"

// GetSchema returns the Auro schema string identified by the id.
// id (int) – the globally unique identifier of the schema.
func (c *Client) GetSchema(subjectID int) (string, error) {
	// # Get the schema for a particular subject id
	// GET /api/proxy-sr/schemas/ids/{int: id}
	path := fmt.Sprintf(schemaPath, subjectID)
	resp, err := c.do(http.MethodGet, path, "", nil, schemaAPIOption)
	if err != nil {
		return "", err
	}

	var res schemaOnlyJSON
	if err = c.readJSON(resp, &res); err != nil {
		return "", err
	}

	return res.Schema, nil
}

// Schema describes a schema, look `GetSchema` for more.
type Schema struct {
	ID int `json:"id,omitempty" yaml:"ID,omitempty"`
	// Name is the name of the schema is registered under.
	Name string `json:"name,omitempty" yaml:"Name"` // Name is the "subject" argument in client-code, this structure is being used on CLI for yaml-file based loading.
	// Version of the returned schema.
	Version int `json:"version"`
	// AvroSchema is the Avro schema string.
	AvroSchema string `json:"schema" yaml:"AvroSchema"`
}

// JSONAvroSchema converts and returns the json form of the "avroSchema" as []byte.
func JSONAvroSchema(avroSchema string) (json.RawMessage, error) {
	var raw json.RawMessage
	err := json.Unmarshal(json.RawMessage(avroSchema), &raw)
	if err != nil {
		return nil, err
	}
	return raw, err
}

// SchemaLatestVersion is the only one valid string for the "versionID", it's the "latest" version string and it's used on `GetLatestSchema`.
const SchemaLatestVersion = "latest"

func checkSchemaVersionID(versionID interface{}) error {
	if versionID == nil {
		return errRequired("versionID (string \"latest\" or int)")
	}

	if verStr, ok := versionID.(string); ok {
		if verStr != SchemaLatestVersion {
			return fmt.Errorf("client: %v string is not a valid value for the versionID input parameter [versionID == \"latest\"]", versionID)
		}
	}

	if verInt, ok := versionID.(int); ok {
		if verInt <= 0 || verInt > 2^31-1 { // it's the max of int32, math.MaxInt32 already but do that check.
			return fmt.Errorf("client: %v integer is not a valid value for the versionID input parameter [ versionID > 0 && versionID <= 2^31-1]", versionID)
		}
	}

	return nil
}

// subject (string) – Name of the subject
// version (versionId [string "latest" or 1,2^31-1]) – Version of the schema to be returned.
// Valid values for versionId are between [1,2^31-1] or the string “latest”.
// The string “latest” refers to the last registered schema under the specified subject.
// Note that there may be a new latest schema that gets registered right after this request is served.
//
// It's not safe to use just an interface to the high-level API, therefore we split this method
// to two, one which will retrieve the latest versioned schema and the other which will accept
// the version as integer and it will retrieve by a specific version.
//
// See `GetLatestSchema` and `GetSchemaAtVersion` instead.
func (c *Client) getSubjectSchemaAtVersion(subject string, versionID interface{}) (s Schema, err error) {
	if subject == "" {
		err = errRequired("subject")
		return
	}

	if err = checkSchemaVersionID(versionID); err != nil {
		return
	}

	// # Get the schema at a particular version
	// GET /api/proxy-sr/subjects/(string: subject)/versions/(versionId: "latest" | int)
	path := fmt.Sprintf(subjectPath+"/versions/%v", subject, versionID)
	resp, respErr := c.do(http.MethodGet, path, "", nil, schemaAPIOption)
	if respErr != nil {
		err = respErr
		return
	}

	err = c.readJSON(resp, &s)
	return
}

// GetLatestSchema returns the latest version of a schema.
// See `GetSchemaAtVersion` to retrieve a subject schema by a specific version.
func (c *Client) GetLatestSchema(subject string) (Schema, error) {
	return c.getSubjectSchemaAtVersion(subject, SchemaLatestVersion)
}

// GetSchemaAtVersion returns a specific version of a schema.
// See `GetLatestSchema` to retrieve the latest schema.
func (c *Client) GetSchemaAtVersion(subject string, versionID int) (Schema, error) {
	return c.getSubjectSchemaAtVersion(subject, versionID)
}

type idOnlyJSON struct {
	ID int `json:"id"`
}

// RegisterSchema registers a schema.
// The returned identifier should be used to retrieve
// this schema from the schemas resource and is different from
// the schema’s version which is associated with that name.
func (c *Client) RegisterSchema(subject string, avroSchema string) (int, error) {
	if subject == "" {
		return 0, errRequired("subject")
	}
	if avroSchema == "" {
		return 0, errRequired("avroSchema")
	}

	schema := schemaOnlyJSON{
		Schema: avroSchema,
	}

	send, err := json.Marshal(schema)
	if err != nil {
		return 0, err
	}

	// # Register a new schema under a particular subject
	// POST /api/proxy-sr/subjects/(string: subject)/versions

	path := fmt.Sprintf(subjectPath+"/versions", subject)
	resp, err := c.do(http.MethodPost, path, contentTypeSchemaJSON, send, schemaAPIOption)
	if err != nil {
		return 0, err
	}

	var res idOnlyJSON
	err = c.readJSON(resp, &res)
	return res.ID, err
}

// deleteSubjectSchemaVersion deletes a specific version of the schema registered under this subject.
// It's being used in `DeleteSchemaVersion` and `DeleteLatestSchemaVersion`.
func (c *Client) deleteSubjectSchemaVersion(subject string, versionID interface{}) (int, error) {
	if subject == "" {
		return 0, errRequired("subject")
	}

	if err := checkSchemaVersionID(versionID); err != nil {
		return 0, err
	}

	// # Delete a particular version of a subject
	// DELETE /api/proxy-sr/subjects/(string: subject)/versions/(versionId: version)
	path := fmt.Sprintf(subjectPath+"/versions/%v", subject, versionID)
	resp, err := c.do(http.MethodDelete, path, contentTypeSchemaJSON, nil, schemaAPIOption)
	if err != nil {
		return 0, err
	}

	var res int
	err = c.readJSON(resp, &res)

	return res, err
}

// DeleteSubjectVersion deletes a specific version of the schema registered under this subject.
// This only deletes the version and the schema id remains intact making it still possible to decode data using the schema id.
// This API is recommended to be used only in development environments or under extreme circumstances where-in,
// its required to delete a previously registered schema for compatibility purposes or re-register previously registered schema.
//
// subject (string) – Name of the subject.
// version (versionId) – Version of the schema to be deleted.
//
// Valid values for versionID are between [1,2^31-1].
// It returns the version (as number) of the deleted schema.
//
// See `DeleteLatestSubjectVersion` too.
func (c *Client) DeleteSubjectVersion(subject string, versionID int) (int, error) {
	return c.deleteSubjectSchemaVersion(subject, versionID)
}

// DeleteLatestSubjectVersion deletes the latest version of the schema registered under this subject.
// This only deletes the version and the schema id remains intact making it still possible to decode data using the schema id.
// This API is recommended to be used only in development environments or under extreme circumstances where-in,
// its required to delete a previously registered schema for compatibility purposes or re-register previously registered schema.
//
// subject (string) – Name of the subject.
//
// It returns the version (as number) of the deleted schema.
//
// See `DeleteSubjectVersion` too.
func (c *Client) DeleteLatestSubjectVersion(subject string) (int, error) {
	return c.deleteSubjectSchemaVersion(subject, SchemaLatestVersion)
}

// CompatibilityLevel describes the valid compatibility levels' type, it's just a string.
// Valid values are:
// `CompatibilityLevelNone`, `CompatibilityLevelFull`, `CompatibilityLevelForward`, `CompatibilityLevelBackward`
// `CompatibilityLevelFullTransitive`, `CompatibilityLevelForwardTransitive`, `CompatibilityLevelBackwardTransitive`.
//
// Read https://docs.confluent.io/current/schema-registry/docs/api.html#compatibility for more.
type CompatibilityLevel string

const (
	// CompatibilityLevelNone is the "NONE" compatibility level.
	CompatibilityLevelNone CompatibilityLevel = "NONE"
	// CompatibilityLevelFull is the "FULL" compatibility level.
	CompatibilityLevelFull CompatibilityLevel = "FULL"
	// CompatibilityLevelForward is the "FORWARD" compatibility level.
	CompatibilityLevelForward CompatibilityLevel = "FORWARD"
	// CompatibilityLevelBackward is the "BACKWARD" compatibility level.
	CompatibilityLevelBackward CompatibilityLevel = "BACKWARD"
	// CompatibilityLevelFullTransitive is the "FULL_TRANSITIVE" compatibility level.
	CompatibilityLevelFullTransitive CompatibilityLevel = "FULL_TRANSITIVE"
	// CompatibilityLevelForwardTransitive is the "FORWARD_TRANSITIVE" compatibility level.
	CompatibilityLevelForwardTransitive CompatibilityLevel = "FORWARD_TRANSITIVE"
	// CompatibilityLevelBackwardTransitive is the "BACKWARD_TRANSITIVE" compatibility level.
	CompatibilityLevelBackwardTransitive CompatibilityLevel = "BACKWARD_TRANSITIVE"
)

// ValidCompatibilityLevels holds a list of the valid compatibility levels,
// see `CompatibilityLevel` type.
var ValidCompatibilityLevels = []CompatibilityLevel{
	CompatibilityLevelNone,
	CompatibilityLevelFull,
	CompatibilityLevelForward,
	CompatibilityLevelBackward,
	CompatibilityLevelFullTransitive,
	CompatibilityLevelForwardTransitive,
	CompatibilityLevelBackwardTransitive,
}

// IsValidCompatibilityLevel checks if a compatibility of string form is a valid compatibility level value.
// See `ValidCompatibilityLevels` too.
func IsValidCompatibilityLevel(compatibility string) bool {
	for _, lv := range ValidCompatibilityLevels {
		if string(lv) == compatibility {
			return true
		}
	}

	return false
}

type (
	compatibilityPutOnlyJSON struct {
		// It can be one of the CompatibilityLevel,
		// NONE, FULL, FORWARD or BACKWARD, FULL_TRANSITIVE, FORWARD_TRANSITIVE or BACKWARD_TRANSITIVE.
		// PUT, for GET its name is "compatibility" so a new struct is created for that, look below.
		Compatibility string `json:"compatibility"`
	}

	compatibilityOnlyJSON struct {
		Compatibility string `json:"compatibilityLevel"`
	}
)

const compatibilityLevelPath = "/api/proxy-sr/config"

// UpdateGlobalCompatibilityLevel sets a new global compatibility level.
// When there are multiple instances of schema registry running in the same cluster,
// the update request will be forwarded to one of the instances designated as the master.
// If the master is not available, the client will get an error code indicating
// that the forwarding has failed.
func (c *Client) UpdateGlobalCompatibilityLevel(level CompatibilityLevel) error {
	lv := compatibilityPutOnlyJSON{
		Compatibility: string(level),
	}

	send, err := json.Marshal(lv)
	if err != nil {
		return err
	}

	// # Update global compatibility level
	// PUT /api/proxy-sr/config
	resp, err := c.do(http.MethodPut, compatibilityLevelPath, contentTypeSchemaJSON, send, schemaAPIOption)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

// GetGlobalCompatibilityLevel returns the global compatibility level,
// "NONE", "FULL", "FORWARD" or "BACKWARD", as described at the `CompatibilityLevel` type.
func (c *Client) GetGlobalCompatibilityLevel() (level CompatibilityLevel, err error) {
	// # Get global compatibility level
	// GET /api/proxy-sr/config
	resp, respErr := c.do(http.MethodGet, compatibilityLevelPath, "", nil, schemaAPIOption)
	if respErr != nil {
		err = respErr
		return
	}

	var levelReq compatibilityOnlyJSON
	err = c.readJSON(resp, &levelReq)
	level = CompatibilityLevel(levelReq.Compatibility)
	return
}

const subjectCompatibilityLevelPath = compatibilityLevelPath + "/%s"

// UpdateSubjectCompatibilityLevel modifies a specific subject(schema)'s compatibility level.
func (c *Client) UpdateSubjectCompatibilityLevel(subject string, level CompatibilityLevel) error {
	if subject == "" {
		return errRequired("subject")
	}

	if string(level) == "" {
		return errRequired("level")
	}

	lv := compatibilityPutOnlyJSON{
		Compatibility: string(level),
	}

	send, err := json.Marshal(lv)
	if err != nil {
		return err
	}

	// # Change compatibility level of a subject
	// PUT /api/proxy-sr/config/(string: subject)
	path := fmt.Sprintf(subjectCompatibilityLevelPath, subject)
	resp, err := c.do(http.MethodPut, path, contentTypeSchemaJSON, send, schemaAPIOption)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

// GetSubjectCompatibilityLevel returns the compatibility level of a specific subject(schema) name.
func (c *Client) GetSubjectCompatibilityLevel(subject string) (level CompatibilityLevel, err error) {
	if subject == "" {
		err = errRequired("subject")
		return
	}

	// # Get compatibility level of a subject
	// GET /api/proxy-sr/config/(string: subject)
	path := fmt.Sprintf(subjectCompatibilityLevelPath, subject)
	resp, respErr := c.do(http.MethodGet, path, "", nil, schemaAPIOption)
	if respErr != nil {
		err = respErr
		return
	}

	var levelReq compatibilityOnlyJSON
	err = c.readJSON(resp, &levelReq)
	level = CompatibilityLevel(levelReq.Compatibility)

	return
}

//
// ACL API
// "ACL" stands for "Access Control Lists".
//

// ACLOperation is a string and it defines the valid operations for ACL.
// See `ACLOperations` to see what operation is valid for a resource type.
type ACLOperation string

const (
	// OpRead is the "Read" ACL operation.
	OpRead ACLOperation = "Read"
	// OpWrite is the "Write" ACL operation.
	OpWrite ACLOperation = "Write"
	// OpDescribe is the "Describe" ACL operation.
	OpDescribe ACLOperation = "Describe"
	// OpDelete is the "Delete" ACL operation.
	OpDelete ACLOperation = "Delete"
	// OpDescribeConfigs is the "DescribeConfigs" ACL operation.
	OpDescribeConfigs ACLOperation = "DescribeConfigs"
	// OpAlterConfigs is the "AlterConfigs" ACL operation.
	OpAlterConfigs ACLOperation = "AlterConfigs"
	// OpAll is the "All" ACL operation.
	OpAll ACLOperation = "All"
	// OpCreate is the "Create" ACL operation.
	OpCreate ACLOperation = "Create"
	// OpClusterAction is the "ClusterAction" ACL operation.
	OpClusterAction ACLOperation = "ClusterAction"
	// OpIdempotentWrite is the "IdempotentWrite" ACL operation.
	OpIdempotentWrite ACLOperation = "IdempotentWrite"
	// OpAlter is the "Alter" ACL operation.
	OpAlter ACLOperation = "Alter"
)

// ACLResourceType is a string and it defines the valid resource types for ACL.
type ACLResourceType string

const (
	// ACLResourceTypeInvalid ACLResourceType = "Invalid"

	// ACLResourceTopic is the "Topic" ACL resource type.
	ACLResourceTopic ACLResourceType = "Topic"
	// ACLResourceGroup is the "Group" ACL resource type.
	ACLResourceGroup ACLResourceType = "Group"
	// ACLResourceCluster is the "Cluster" ACL resource type.
	ACLResourceCluster ACLResourceType = "Cluster"
	// ACLResourceTransactionalID is the "TransactionalId" ACL resource type.
	ACLResourceTransactionalID ACLResourceType = "TransactionalId"
)

// ACLOperations is a map which contains the allowed ACL operations(values) per resource type(key).
var ACLOperations = map[ACLResourceType][]ACLOperation{
	ACLResourceTopic:           {OpRead, OpWrite, OpDescribe, OpDelete, OpDescribeConfigs, OpAlterConfigs, OpAll},
	ACLResourceGroup:           {OpRead, OpDescribe, OpAll},
	ACLResourceCluster:         {OpCreate, OpClusterAction, OpDescribeConfigs, OpAlterConfigs, OpIdempotentWrite, OpAlter, OpDescribe, OpAll},
	ACLResourceTransactionalID: {OpDescribe, OpWrite, OpAll},
}

func (op ACLOperation) isValidForResourceType(resourceType ACLResourceType) bool {
	operations, has := ACLOperations[resourceType]
	if !has {
		return false
	}

	for _, operation := range operations {
		if operation == op {
			return true
		}
	}

	return false
}

// ACLPermissionType is a string and it defines the valid permission types for ACL.
type ACLPermissionType string

const (
	// ACLPermissionAllow is the "Allow" ACL permission type.
	ACLPermissionAllow ACLPermissionType = "Allow"
	// ACLPermissionDeny is the "Deny" ACL permission type.
	ACLPermissionDeny ACLPermissionType = "Deny"
)

// ACL is the type which defines a single Apache Access Control List.
type ACL struct {
	ResourceType   ACLResourceType   `json:"resourceType" yaml:"ResourceType"`     // required.
	ResourceName   string            `json:"resourceName" yaml:"ResourceName"`     // required.
	Principal      string            `json:"principal" yaml:"Principal"`           // required.
	PermissionType ACLPermissionType `json:"permissionType" yaml:"PermissionType"` // required.
	Host           string            `json:"host" yaml:"Host"`                     // required.
	Operation      ACLOperation      `json:"operation" yaml:"Operation"`           // required.
}

// Validate force validates the acl's resource type, permission type and operation.
// It returns an error if the operation is not valid for the resource type.
func (acl *ACL) Validate() error {
	// upper the first letter here on the resourceType, permissionType and operation before any action.
	acl.ResourceType = ACLResourceType(strings.Title(string(acl.ResourceType)))
	acl.PermissionType = ACLPermissionType(strings.Title(string(acl.PermissionType)))
	acl.Operation = ACLOperation(strings.Title(string(acl.Operation)))

	if !acl.Operation.isValidForResourceType(acl.ResourceType) {
		validOps := ACLOperations[acl.ResourceType]
		errMsg := ""
		if validOps == nil {
			errMsg = fmt.Sprintf("invalid resource type. Valid resource types are: '%s', '%s', '%s' or '%s'",
				ACLResourceTopic, ACLResourceGroup, ACLResourceCluster, ACLResourceTransactionalID)
		} else {
			errMsg = fmt.Sprintf("invalid operation for resource type: '%s'. The valid operations for this type are: %s", acl.ResourceType, validOps)
		}

		return fmt.Errorf(errMsg)
	}

	if acl.Host == "" {
		acl.Host = "*" // wildcard, all.
	}

	return nil
}

const aclPath = "/api/acl"

// CreateOrUpdateACL sets an Apache Kafka Access Control List.
// Use the defined types when needed, example:
// `client.CreateOrUpdateACL(lenses.ACL{lenses.ACLResourceTopic, "transactions", "principalType:principalName", lenses.ACLPermissionAllow, "*", lenses.OpRead})`
//
// Note that on the "host" input argument you should use IP addresses as domain names are not supported at the moment by Apache Kafka.
func (c *Client) CreateOrUpdateACL(acl ACL) error {
	if err := acl.Validate(); err != nil {
		return err
	}

	send, err := json.Marshal(acl)
	if err != nil {
		return err
	}

	resp, err := c.do(http.MethodPut, aclPath, contentTypeJSON, send)
	if err != nil {
		return err
	}

	// note: the status code errors are checked in the `do` on every request.
	return resp.Body.Close()
}

// GetACLs returns all the available Apache Kafka Access Control Lists.
func (c *Client) GetACLs() ([]ACL, error) {
	resp, err := c.do(http.MethodGet, aclPath, "", nil)
	if err != nil {
		return nil, err
	}

	var acls []ACL
	err = c.readJSON(resp, &acls)
	return acls, err
}

// DeleteACL deletes an existing Apache Kafka Access Control List.
func (c *Client) DeleteACL(acl ACL) error {
	if err := acl.Validate(); err != nil {
		return err
	}

	send, err := json.Marshal(acl)
	if err != nil {
		return err
	}

	resp, err := c.do(http.MethodDelete, aclPath, contentTypeJSON, send)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

//
// Quota API
//

// QuotaEntityType is a string and it defines the valid entity types for a single Quota.
type QuotaEntityType string

const (
	// QuotaEntityClient is the "CLIENT" Quota entity type.
	QuotaEntityClient QuotaEntityType = "CLIENT"
	// QuotaEntityClients is the "CLIENTS" Quota entity type.
	QuotaEntityClients QuotaEntityType = "CLIENTS"
	// QuotaEntityClientsDefault is the "CLIENTS DEFAULT" Quota entity type.
	QuotaEntityClientsDefault QuotaEntityType = "CLIENTS DEFAULT"
	// QuotaEntityUser is the "USER" Quota entity type.
	QuotaEntityUser QuotaEntityType = "USER"
	// QuotaEntityUsers is the "USERS" Quota entity type.
	QuotaEntityUsers QuotaEntityType = "USERS"
	// QuotaEntityUserClient is the "USERCLIENT" Quota entity type.
	QuotaEntityUserClient QuotaEntityType = "USERCLIENT"
	// QuotaEntityUsersDefault is the "USERS DEFAULT" Quota entity type.
	QuotaEntityUsersDefault QuotaEntityType = "USERS DEFAULT"
)

type (
	// Quota is the type which defines a single Quota.
	Quota struct {
		// EntityType can be either `QuotaEntityClient`, `QuotaEntityClients`,
		// `QuotaEntityClientsDefault`, `QuotaEntityUser`, `QuotaEntityUsers`, `QuotaEntityUserClient`
		// or `QuotaEntityUsersDefault`.
		EnityType QuotaEntityType `json:"entityType" yaml:"EntityType"`
		// Entityname is the Kafka client id for "CLIENT"
		// and "CLIENTS" and user name for "USER", "USER" and "USERCLIENT", the `QuotaEntityXXX`.
		EntityName string `json:"entityName" yaml:"EntityName"`
		// Child is optional and only present for entityType `QuotaEntityUserClient` and is the client id.
		Child string `json:"child,omitempty" yaml:"Child"`
		// Properties  is a map of the quota constraints, the `QuotaConfig`.
		Properties QuotaConfig `json:"properties" yaml:"Properties"`
		// URL is the url from this quota in Lenses.
		URL string `json:"url" yaml:"URL"`

		IsAuthorized bool `json:"isAuthorized" yaml:"IsAuthorized"`
	}

	// QuotaConfig is a typed struct which defines the
	// map of the quota constraints, producer_byte_rate, consumer_byte_rate and request_percentage.
	QuotaConfig struct {
		ProducerByteRate  string `json:"producer_byte_rate" yaml:"ProducerByteRate"`
		ConsumerByteRate  string `json:"consumer_byte_rate" yaml:"ConsumerByteRate"`
		RequestPercentage string `json:"request_percentage" yaml:"RequestPercentage"`
	}
)

const quotasPath = "/api/quotas"

// GetQuotas returns a list of all available quotas.
func (c *Client) GetQuotas() ([]Quota, error) {
	resp, err := c.do(http.MethodGet, quotasPath, "", nil)
	if err != nil {
		return nil, err
	}

	var quotas []Quota
	err = c.readJSON(resp, &quotas)
	return quotas, err
}

// /api/quotas/users
const quotasPathAllUsers = quotasPath + "/users"

// CreateOrUpdateQuotaForAllUsers sets the default quota for all users.
// Read more at: http://lenses.stream/using-lenses/user-guide/quotas.html.
func (c *Client) CreateOrUpdateQuotaForAllUsers(config QuotaConfig) error {
	send, err := json.Marshal(config)
	if err != nil {
		return err
	}

	resp, err := c.do(http.MethodPut, quotasPathAllUsers, contentTypeJSON, send)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

// DeleteQuotaForAllUsers deletes the default for all users.
// Read more at: http://lenses.stream/using-lenses/user-guide/quotas.html.
func (c *Client) DeleteQuotaForAllUsers() error {
	resp, err := c.do(http.MethodDelete, quotasPathAllUsers, "", nil)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

// /api/quotas/users/{user}
const quotasPathUser = quotasPathAllUsers + "/%s"

// CreateOrUpdateQuotaForUser sets a quota for a user.
// Read more at: http://lenses.stream/using-lenses/user-guide/quotas.html.
func (c *Client) CreateOrUpdateQuotaForUser(user string, config QuotaConfig) error {
	send, err := json.Marshal(config)
	if err != nil {
		return err
	}

	path := fmt.Sprintf(quotasPathUser, user)
	resp, err := c.do(http.MethodPut, path, contentTypeJSON, send)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

// DeleteQuotaForUser deletes a quota for a user.
func (c *Client) DeleteQuotaForUser(user string) error {
	path := fmt.Sprintf(quotasPathUser, user)
	resp, err := c.do(http.MethodDelete, path, "", nil)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

// /api/quotas/users/{user}/clients
const quotasPathUserAllClients = quotasPathUser + "/clients"

// CreateOrUpdateQuotaForUserAllClients sets a quota for a user for all clients.
// Read more at: http://lenses.stream/using-lenses/user-guide/quotas.html.
func (c *Client) CreateOrUpdateQuotaForUserAllClients(user string, config QuotaConfig) error {
	send, err := json.Marshal(config)
	if err != nil {
		return err
	}

	path := fmt.Sprintf(quotasPathUserAllClients, user)
	resp, err := c.do(http.MethodPut, path, contentTypeJSON, send)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

// DeleteQuotaForUserAllClients deletes for all client ids for a user.
func (c *Client) DeleteQuotaForUserAllClients(user string) error {
	path := fmt.Sprintf(quotasPathUserAllClients, user)
	resp, err := c.do(http.MethodDelete, path, "", nil)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

// /api/quotas/users/{user}/clients/{client-id}
const quotasPathUserClient = quotasPathUserAllClients + "/%s"

// CreateOrUpdateQuotaForUserClient sets the quota for a user/client pair.
// Read more at: http://lenses.stream/using-lenses/user-guide/quotas.html.
func (c *Client) CreateOrUpdateQuotaForUserClient(user, clientID string, config QuotaConfig) error {
	send, err := json.Marshal(config)
	if err != nil {
		return err
	}

	path := fmt.Sprintf(quotasPathUserClient, user, clientID)
	resp, err := c.do(http.MethodPut, path, contentTypeJSON, send)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

// DeleteQuotaForUserClient deletes the quota for a user/client pair.
func (c *Client) DeleteQuotaForUserClient(user, clientID string) error {
	path := fmt.Sprintf(quotasPathUserClient, user, clientID)
	resp, err := c.do(http.MethodDelete, path, "", nil)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

// /api/quotas/clients
const quotasPathAllClients = quotasPath + "/clients"

// CreateOrUpdateQuotaForAllClients sets the default quota for all clients.
// Read more at: http://lenses.stream/using-lenses/user-guide/quotas.html.
func (c *Client) CreateOrUpdateQuotaForAllClients(config QuotaConfig) error {
	send, err := json.Marshal(config)
	if err != nil {
		return err
	}

	resp, err := c.do(http.MethodPut, quotasPathAllClients, contentTypeJSON, send)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

// DeleteQuotaForAllClients deletes the default quota for all clients.
func (c *Client) DeleteQuotaForAllClients() error {
	resp, err := c.do(http.MethodDelete, quotasPathAllClients, "", nil)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

// /api/quotas/clients/{client-id}
const quotasPathClient = quotasPathAllClients + "/%s"

// CreateOrUpdateQuotaForClient sets the quota for a specific client.
// Read more at: http://lenses.stream/using-lenses/user-guide/quotas.html.
func (c *Client) CreateOrUpdateQuotaForClient(clientID string, config QuotaConfig) error {
	send, err := json.Marshal(config)
	if err != nil {
		return err
	}

	path := fmt.Sprintf(quotasPathClient, clientID)
	resp, err := c.do(http.MethodPut, path, contentTypeJSON, send)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

// DeleteQuotaForClient deletes quotas for a client id.
func (c *Client) DeleteQuotaForClient(clientID string) error {
	path := fmt.Sprintf(quotasPathClient, clientID)
	resp, err := c.do(http.MethodDelete, path, "", nil)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

// Alert API

type (
	// AlertSetting describes the type of list entry of the `GetAlertSetting` and `CreateOrUpdateAlertSettingCondition`.
	AlertSetting struct {
		ID                int               `json:"id"`
		Description       string            `json:"description"`
		Category          string            `json:"category"`
		Enabled           bool              `json:"enabled"`
		Docs              string            `json:"docs,omitempty"`
		ConditionTemplate string            `json:"conditionTemplate,omitempty"`
		ConditionRegex    string            `json:"conditionRegex,omitempty"`
		Conditions        map[string]string `json:"conditions,omitempty"`
		IsAvailable       bool              `json:"isAvailable"`
	}

	// AlertSettings describes the type of list entry of the `GetAlertSettings`.
	AlertSettings struct {
		Categories AlertSettingsCategoryMap `json:"categories"`
	}

	AlertSettingsCategoryMap struct {
		Infrastructure []AlertSetting `json:"Infrastructure"`
		Consumers      []AlertSetting `json:"Consumers"`
	}
)

const (
	alertsPath                 = "/api/alerts"
	alertSettingsPath          = alertsPath + "/settings"
	alertSettingPath           = alertSettingsPath + "/%d"
	alertSettingConditionsPath = alertSettingPath + "/condition"
	alertSettingConditionPath  = alertSettingConditionsPath + "/%s" // UUID for condition.
)

// GetAlertSettings returns all the configured alert settings.
// Alerts are divided into two categories:
//
// * Infrastructure - These are out of the box alerts that be toggled on and offset.
// * Consumer group - These are user-defined alerts on consumer groups.
//
// Alert notifications are the result of an `AlertSetting` Condition being met on an `AlertSetting`.
func (c *Client) GetAlertSettings() (AlertSettings, error) {
	resp, err := c.do(http.MethodGet, alertSettingsPath, "", nil)
	if err != nil {
		return AlertSettings{}, err
	}

	var settings AlertSettings
	err = c.readJSON(resp, &settings)
	return settings, err
}

// GetAlertSetting returns a specific alert setting based on its "id".
func (c *Client) GetAlertSetting(id int) (setting AlertSetting, err error) {
	path := fmt.Sprintf(alertSettingPath, id)
	resp, respErr := c.do(http.MethodGet, path, "", nil)
	if respErr != nil {
		err = respErr
		return
	}

	err = c.readJSON(resp, &setting)
	return
}

// EnableAlertSetting enables a specific alert setting based on its "id".
func (c *Client) EnableAlertSetting(id int) error {
	path := fmt.Sprintf(alertSettingPath, id)
	resp, err := c.do(http.MethodPut, path, "", nil)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

// AlertSettingConditions map with UUID as key and the condition as value, used on `GetAlertSettingConditions`.
type AlertSettingConditions map[string]string

// GetAlertSettingConditions returns alert setting's conditions as a map of strings.
func (c *Client) GetAlertSettingConditions(id int) (AlertSettingConditions, error) {
	path := fmt.Sprintf(alertSettingConditionsPath, id)
	resp, err := c.do(http.MethodGet, path, "", nil)
	if err != nil {
		return nil, err
	}

	var conds AlertSettingConditions
	if err = c.readJSON(resp, &conds); err != nil {
		return nil, err
	}
	return conds, nil
}

type (
	// Alert is the request payload that is used to register an Alert via `RegisterAlert` and the response that client retrieves from the `GetAlerts`.
	Alert struct {
		// AlertID  is a unique identifier for the setting corresponding to this alert. See the available ids via `GetAlertSettings`.
		AlertID int `json:"alertId" yaml:"AlertID"`
		// EndsAt is the time as string the alert ended at.
		EndsAt string `json:"endsAt" yaml:"EndsAt"`
		// StartsAt is the time as string, in ISO format, for when the alert starts
		StartsAt string `json:"startsAt" yaml:"StartsAt"`
		// Labels field is a list of key-value pairs. It must contain a non empty `Severity` value.
		Labels AlertLabels `json:"labels" yaml:"Labels"`
		// Annotations is a list of key-value pairs. It contains the summary, source, and docs fields.
		Annotations AlertAnnotations `json:"annotations" yaml:"Annotations"`
		// GeneratorURL is a unique URL identifying the creator of this alert.
		// It matches AlertManager requirements for providing this field.
		GeneratorURL string `json:"generatorURL" yaml:"GeneratorURL"`
	}

	// AlertLabels labels for the `Alert`, at least Severity should be filled.
	AlertLabels struct {
		Category string `json:"category,omitempty" yaml:"Category,omitempty"`
		Severity string `json:"severity" yaml:"Severity,omitempty"`
		Instance string `json:"instance,omitempty" yaml:"Instance,omitempty"`
	}

	// AlertAnnotations annotations for the `Alert`, at least Summary should be filled.
	AlertAnnotations struct {
		Summary string `json:"summary" yaml:"Summary"`
		Source  string `json:"source,omitempty" yaml:"Source,omitempty"`
		Docs    string `json:"docs,omitempty" yaml:"Docs,omitempty"`
	}
)

// RegisterAlert registers an Alert, returns an error on failure.
func (c *Client) RegisterAlert(alert Alert) error {
	if alert.Labels.Severity == "" {
		return errRequired("Labels.Severity")
	}

	alert.Labels.Severity = strings.ToUpper(alert.Labels.Severity)

	send, err := json.Marshal(alert)
	if err != nil {
		return err
	}

	resp, err := c.do(http.MethodPost, alertsPath, contentTypeJSON, send)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

// GetAlerts returns the registered alerts.
func (c *Client) GetAlerts() (alerts []Alert, err error) {
	resp, respErr := c.do(http.MethodGet, alertsPath, "", nil)
	if respErr != nil {
		err = respErr
		return
	}

	err = c.readJSON(resp, &alerts)
	return
}

// CreateOrUpdateAlertSettingCondition sets a condition(expression text) for a specific alert setting.
func (c *Client) CreateOrUpdateAlertSettingCondition(alertSettingID int, condition string) error {
	path := fmt.Sprintf(alertSettingConditionsPath, alertSettingID)
	resp, err := c.do(http.MethodPost, path, "text/plain", []byte(condition))
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

// DeleteAlertSettingCondition deletes a condition from an alert setting.
func (c *Client) DeleteAlertSettingCondition(alertSettingID int, conditionUUID string) error {
	path := fmt.Sprintf(alertSettingConditionPath, alertSettingID, conditionUUID)
	resp, err := c.do(http.MethodDelete, path, "", nil)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

const (
	alertsPathSSE       = "/api/sse/alerts"
	alertsSSEDataPrefix = "data:"
)

// AlertHandler is the type of func that can be registered to receive alerts via the `GetAlertsLive`.
type AlertHandler func(Alert) error

// GetAlertsLive receives alert notifications in real-time from the server via a Send Server Event endpoint.
func (c *Client) GetAlertsLive(handler AlertHandler) error {
	resp, err := c.do(http.MethodGet, alertsPathSSE, contentTypeJSON, nil, func(r *http.Request) {
		r.Header.Add(acceptHeaderKey, "application/json, text/event-stream")
	}, schemaAPIOption)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	reader, err := c.acquireResponseBodyStream(resp)
	if err != nil {
		return err
	}

	streamReader := bufio.NewReader(reader)

	for {
		line, err := streamReader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return nil // we read until the the end, exit with no error here.
			}
			return err // exit on first failure.
		}

		// ignore all except data: ..., heartbeats.
		if len(line) < len(alertsSSEDataPrefix)+1 {
			continue
		}

		message := line[len(alertsSSEDataPrefix):] // we need everything after the 'data:'.

		// it can return data:[empty here] when it stops, let's stop it
		if len(message) < 2 {
			return nil // stop here for now.
		}

		alert := Alert{}

		if err = json.Unmarshal(message, &alert); err != nil {
			// exit on first error here as well.
			return err
		}

		if err = handler(alert); err != nil {
			return err // stop on first error by the caller.
		}
	}
}
