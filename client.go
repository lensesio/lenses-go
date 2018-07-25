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

// User represents the user of the client.
type User struct {
	Token                string   `json:"token"`
	Name                 string   `json:"user" header:"Name"`
	SchemaRegistryDelete bool     `json:"schemaRegistryDelete" header:"Schema Registry Delete"`
	Roles                []string `json:"roles" header:"Roles"`
}

// Client is the lenses http client.
// It contains the necessary API calls to communicate and develop via lenses.
type Client struct {
	Config     *ClientConfig
	configFull *Config // not exported, used for `ConnectionOptions`.
	// PersistentRequestModifier can be used to modify the *http.Request before send it to the backend.
	PersistentRequestModifier RequestOption

	// Progress                  func(current, total int64)
	// User is generated on `lenses#OpenConnection` function based on the `Config#Authentication`.
	User User

	// the client is created on the `lenses#OpenConnection` function, it can be customized via options there.
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

// isAuthorized is called inside the `Client#Do` and it closes the body reader if no accessible.
// 401	Unauthorized	[RFC7235, Section 3.1]
func isAuthorized(resp *http.Response) bool { return resp.StatusCode != http.StatusUnauthorized }

// isOK is called inside the `Client#Do` and it closes the body reader if no accessible.
func isOK(resp *http.Response) bool {
	return resp.StatusCode == http.StatusOK ||
		resp.StatusCode == http.StatusCreated || /* see CreateOrUpdateConnector for the `StatusCreated` */
		resp.StatusCode == http.StatusAccepted || /* see PauseConnector for the `StatusAccepted` */
		(resp.Request.Method == http.MethodDelete && resp.StatusCode == http.StatusNoContent) || /* see RemoveConnector for the `StatusNoContnet` */
		(resp.Request.Method == http.MethodPost && resp.StatusCode == http.StatusNoContent) || /* see Restart tasks for the `StatusNoContnet` */
		(resp.StatusCode == http.StatusBadRequest && resp.Request.Method == http.MethodGet) || /* for things like LSQL which can return 400 if invalid query, we need to read the json and print the error message */
		(resp.Request.Method == http.MethodDelete && ((resp.StatusCode == http.StatusForbidden) || (resp.StatusCode == http.StatusBadRequest))) /* for things like deletion if not proper user access or invalid value of something passed */
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
var ErrCredentialsMissing = fmt.Errorf("credentials missing or invalid")

// RequestOption is just a func which receives the current HTTP request and alters it,
// if the return value of the error is not nil then `Client#Do` fails with that error.
type RequestOption func(r *http.Request) error

var schemaAPIOption = func(r *http.Request) error {
	r.Header.Add(acceptHeaderKey, contentTypeSchemaJSON)
	return nil
}

// ResourceError is being fired from all API calls when an error code is received.
type ResourceError struct {
	StatusCode int    `json:"statusCode" header:"Status Code"`
	Method     string `json:"method" header:"Method"`
	URI        string `json:"uri" header:"Target"`
	Body       string `json:"message" header:"Message"`
}

// Error returns the detailed cause of the error.
func (err ResourceError) Error() string {
	return fmt.Sprintf("client: (%s: %s) failed with status code %d%s",
		err.Method, err.URI, err.StatusCode, err.Body)
}

// Code returns the status code.
func (err ResourceError) Code() int {
	return err.StatusCode
}

// Message returns the message of the error or the whole body if it's unknown error.
func (err ResourceError) Message() string {
	return err.Body
}

// NewResourceError is just a helper to create a new `ResourceError` to return from custom calls, it's "cli-compatible".
func NewResourceError(statusCode int, uri, method, body string) ResourceError {
	unescapedURI, _ := url.QueryUnescape(uri)

	return ResourceError{
		StatusCode: statusCode,
		URI:        unescapedURI,
		Method:     method,
		Body:       body,
	}
}

// Do is the lower level of a client call, manually sends an HTTP request to the lenses box backend based on the `Client#Config`
// and returns an HTTP response.
func (c *Client) Do(method, path, contentType string, send []byte, options ...RequestOption) (*http.Response, error) {
	if path[0] == '/' { // remove beginning slash, if any.
		path = path[1:]
	}

	uri := c.Config.Host + "/" + path

	golog.Debugf("Client#Do.req:\n\turi: %s:%s\n\tsend: %s", method, uri, string(send))

	req, err := http.NewRequest(method, uri, acquireBuffer(send))
	if err != nil {
		return nil, err
	}
	// before sending requests here.

	// set the token header.
	if c.Config.Token != "" {
		req.Header.Set(xKafkaLensesTokenHeaderKey, c.Config.Token)
	}

	// set the content type if any.
	if contentType != "" {
		req.Header.Set(contentTypeHeaderKey, contentType)
	}

	// response accept gziped content.
	req.Header.Add(acceptEncodingHeaderKey, gzipEncodingHeaderValue)

	if c.PersistentRequestModifier != nil {
		if err := c.PersistentRequestModifier(req); err != nil {
			return nil, err
		}
	}

	for _, opt := range options {
		if err = opt(req); err != nil {
			return nil, err
		}
	}

	// here will print all the headers, including the token (because it may be useful for debugging)
	// --so bug reporters should be careful here to invalidate the token after that.
	golog.Debugf("Client#Do.req.Headers: %#+v", req.Header)

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
		defer resp.Body.Close()
		var errBody string

		if strings.Contains(resp.Header.Get(contentTypeHeaderKey), "text/html") {
			// if the body is html, then don't read it, it doesn't contain the raw info we need.
		} else {
			// else give the whole body to the error context.
			b, err := c.ReadResponseBody(resp)
			if err != nil {
				errBody = " unable to read body: " + err.Error()
			} else {
				errBody = "\n" + string(b)
			}
		}

		return nil, NewResourceError(resp.StatusCode, uri, method, errBody)
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

const bufN = 512

// var errEmptyResponse = fmt.Errorf("")

// ReadResponseBody is the lower-level method of client to read the result of a `Client#Do`, it closes the body stream.
//
// See `ReadJSON` too.
func (c *Client) ReadResponseBody(resp *http.Response) ([]byte, error) {
	reader, err := c.acquireResponseBodyStream(resp)
	if err != nil {
		return nil, err
	}

	/*
			var body []byte

			if c.Progress != nil {
				var (
					total   = resp.ContentLength
					current int64
				)

				for {
					buf := make([]byte, bufN)
					readen, readErr := reader.Read(buf)
					// Callers should always process the n > 0 bytes returned before
					// considering the error err. Doing so correctly handles I/O errors
					// that happen after reading some bytes and also both of the
					// allowed EOF behaviors.
					if readen > 0 {
						current += int64(readen)
						body = append(body, buf[:readen]...)
						c.Progress(current, total) // call it every x ms or let the user decide?
					}
					if readErr != nil {
						if readErr == io.EOF {
							break
						}

						return nil, readErr
					}
				}
			} else {
				body, err = ioutil.ReadAll(reader)
		    }
	*/

	b, err := ioutil.ReadAll(reader)
	if err = reader.Close(); err != nil {
		return nil, err
	}

	// if len(b) == 0 || (len(b) == 2 && (b[0] == '[' && b[1] == ']') || (b[0] == '{' && b[1] == '}')) {
	// 	return nil, errEmptyResponse
	// }

	if c.Config.Debug {
		rawBodyString := string(b)
		// print both body and error, because both of them may be formated by the `readResponseBody`'s caller.
		golog.Debugf("Client#Do.resp:\n\tbody: %s\n\tstatus code: %d\n\terror: %v", rawBodyString, resp.StatusCode, err)
	}

	// return the body.
	return b, err
}

// ReadJSON is one of the lower-level methods of the client to read the result of a `Client#Do`, it closes the body stream.
//
// See `ReadResponseBody` lower-level of method to read a response for more details.
func (c *Client) ReadJSON(resp *http.Response, valuePtr interface{}) error {
	b, err := c.ReadResponseBody(resp)
	if err != nil {
		return err
	}

	err = json.Unmarshal(b, valuePtr)
	if c.Config.Debug {
		if syntaxErr, ok := err.(*json.SyntaxError); ok {
			golog.Errorf("Client#ReadJSON: syntax error at offset %d: %s", syntaxErr.Offset, syntaxErr.Error())
		}
	}
	return err
}

// GetAccessToken returns the access token that
// generated from the `OpenConnection` or given by the configuration.
func (c *Client) GetAccessToken() string {
	return c.Config.Token
}

const logoutPath = "api/logout?token="

// Logout invalidates the token and revoke its access.
// A new Client, using `OpenConnection`, should be created in order to continue after this call.
func (c *Client) Logout() error {
	if c.Config.Token == "" {
		return ErrCredentialsMissing
	}

	path := logoutPath + c.Config.Token
	resp, err := c.Do(http.MethodGet, path, "", nil)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

// LicenseInfo describes the data received from the `GetLicenseInfo`.
type LicenseInfo struct {
	ClientID    string `json:"clientId" header:"ID,text"`
	IsRespected bool   `json:"isRespected" header:"Respected"`
	MaxBrokers  int    `json:"maxBrokers" header:"Max Brokers"`
	MaxMessages int    `json:"maxMessages,omitempty" header:"/ Messages"`
	Expiry      int64  `json:"expiry" header:"Expires"`

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

const licensePath = "api/license"

// GetLicenseInfo returns the license information for the connected lenses box.
func (c *Client) GetLicenseInfo() (LicenseInfo, error) {
	var lc LicenseInfo

	resp, err := c.Do(http.MethodGet, licensePath, "", nil)
	if err != nil {
		return lc, err
	}

	if err = c.ReadJSON(resp, &lc); err != nil {
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
	configPath = "api/config"
)

// GetConfig returns the whole configuration of the lenses box,
// which can be changed from box to box and it's read-only,
// therefore it returns a map[string]interface{} based on the
// json response body.
//
// To retrieve the execution mode of the box with safety,
// see the `Client#GetExecutionMode` instead.
func (c *Client) GetConfig() (map[string]interface{}, error) {
	resp, err := c.Do(http.MethodGet, configPath, "", nil, func(r *http.Request) error {
		r.Header.Set("Accept", "application/json, text/plain")
		return nil
	})

	if err != nil {
		return nil, err
	}

	data := make(map[string]interface{}, 0) // maybe make those statically as well, we'll see.
	if err = c.ReadJSON(resp, &data); err != nil {
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
	Name     string `json:"name" header:"Name"`
	URL      string `json:"url"` //header:"URL"`
	Statuses string `json:"statuses" header:"Status"`
	Config   string `json:"config" header:"Config"`
	Offsets  string `json:"offsets" header:"Offsets,count"`
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
	resp, respErr := c.Do(http.MethodGet, path, contentTypeJSON, nil)
	if respErr != nil {
		err = respErr
		return
	}

	err = c.ReadJSON(resp, &v)
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
		IsTimeRemaining bool `json:"isTimeRemaining" header:"Time Remaining"`
		// If true there was no more data on the topic and `max.zero.polls` was reached.
		IsTopicEnd bool `json:"isTopicEnd" header:"End"`
		// If true the query has been stopped by admin  (Cancel query equivalence).
		IsStopped bool `json:"isStopped" header:"Stopped"`
		// Number of records read from Kafka.
		TotalRecords int `json:"totalRecords" header:"Total /"`
		// Number of records not matching the filter.
		SkippedRecords int `json:"skippedRecords" header:"Skipped Records"`
		// Max number of records to pull (driven by LIMIT X,
		// if LIMIT is not present it gets the default config in LENSES).
		RecordsLimit int `json:"recordsLimit" header:"Records Limit"`
		// Total size in bytes read from Kafka.
		TotalSizeRead int64 `json:"totalSizeRead" header:"Total Size Read"`
		// Total size in bytes (Kafka size) for the records.
		Size int64 `json:"size" header:"Size"`
		// The topic offsets.
		// If query parameter `&offsets=true` is not present it won't pull the details.
		Offsets []LSQLOffset `json:"offsets" header:"Offsets,count"`
	}

	// LSQLOffset the form of the offset record data that LSQL call returns once.
	LSQLOffset struct {
		Partition int   `json:"partition" header:"Partition"`
		Min       int64 `json:"min" header:"Min"`
		Max       int64 `json:"max" header:"Max"`
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
	resp, err := c.Do(http.MethodGet, path, contentTypeJSON, nil, func(r *http.Request) error {
		r.Header.Add(acceptHeaderKey, "application/json, text/event-stream")
		return nil
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
	ID        int64  `json:"id" header:"ID" header:"ID,text"`
	SQL       string `json:"sql" header:"SQL" header:"SQL"`
	User      string `json:"user" header:"User" header:"User"`
	Timestamp int64  `json:"ts" header:"Timestamp" header:"Timestamp"`
}

// GetRunningQueries returns a list of the current sql running queries.
func (c *Client) GetRunningQueries() ([]LSQLRunningQuery, error) {
	resp, err := c.Do(http.MethodGet, queriesPath, "", nil)
	if err != nil {
		return nil, err
	}

	var queries []LSQLRunningQuery
	err = c.ReadJSON(resp, &queries)
	return queries, err
}

// CancelQuery stops a running query based on its ID.
// It returns true whether it was cancelled otherwise false or/and error.
func (c *Client) CancelQuery(id int64) (bool, error) {
	path := fmt.Sprintf(queriesPath+"/%d", id)
	resp, err := c.Do(http.MethodDelete, path, "", nil)
	if err != nil {
		return false, err
	}

	var canceled bool
	err = c.ReadJSON(resp, &canceled)
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
	resp, respErr := c.Do(http.MethodGet, topicsPath, "", nil)
	if respErr != nil {
		err = respErr
		return
	}

	err = c.ReadJSON(resp, &topics)
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

const topicsAvailableConfigKeysPath = topicsPath + "/availableConfigKeys"

// GetAvailableTopicConfigKeys retrieves a list of available configs for topics.
func (c *Client) GetAvailableTopicConfigKeys() ([]string, error) {
	resp, err := c.Do(http.MethodGet, topicsAvailableConfigKeysPath, "", nil)
	if err != nil {
		return nil, err
	}

	var keys []string
	if err = c.ReadJSON(resp, &keys); err != nil {
		return nil, err
	}

	return keys, nil
}

type (
	// TopicMetadata describes the data received from the `GetTopicsMetadata`
	// and the payload to send on the `CreateTopicMetadata`.
	TopicMetadata struct {
		TopicName string `json:"topicName" yaml:"TopicName" header:"Topic"`
		KeyType   string `json:"keyType,omitempty" yaml:"KeyType" header:"Key /,NULL"`
		ValueType string `json:"valueType,omitempty" yaml:"ValueType" header:"Value Type,NULL"`

		ValueSchemaRaw string `json:"valueSchema,omitempty" yaml:"ValueSchema,omitempty"` // for response read.
		KeySchemaRaw   string `json:"keySchema,omitempty" yaml:"KeySchema,omitempty"`     // for response read.
	}

	/*
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
	*/
)

const (
	topicsMetadataPath = "api/metadata/topics"
	topicMetadataPath  = topicsMetadataPath + "/%s"
)

// GetTopicsMetadata retrieves and returns all the topics' available metadata.
func (c *Client) GetTopicsMetadata() ([]TopicMetadata, error) {
	resp, err := c.Do(http.MethodGet, topicsMetadataPath, "", nil)
	if err != nil {
		return nil, err
	}

	var meta []TopicMetadata

	err = c.ReadJSON(resp, &meta)
	return meta, err
}

// GetTopicMetadata retrieves and returns a topic's metadata.
func (c *Client) GetTopicMetadata(topicName string) (TopicMetadata, error) {
	var meta TopicMetadata

	if topicName == "" {
		return meta, errRequired("topicName")
	}

	path := fmt.Sprintf(topicMetadataPath, topicName)
	resp, err := c.Do(http.MethodGet, path, "", nil)
	if err != nil {
		return meta, err
	}

	err = c.ReadJSON(resp, &meta)
	return meta, err
}

// CreateOrUpdateTopicMetadata adds or updates an existing topic metadata.
func (c *Client) CreateOrUpdateTopicMetadata(metadata TopicMetadata) error {
	if metadata.TopicName == "" {
		return errRequired("metadata.TopicName")
	}

	path := fmt.Sprintf(topicMetadataPath, metadata.TopicName)
	path += fmt.Sprintf("?keyType=%s&valueType=%s", metadata.KeyType, metadata.ValueType) // required.

	// optional.
	if len(metadata.KeySchemaRaw) > 0 {
		path += "&keySchema=" + string(metadata.KeySchemaRaw)
	}

	if len(metadata.ValueSchemaRaw) > 0 {
		path += "&valueSchema" + string(metadata.ValueSchemaRaw)
	}

	resp, err := c.Do(http.MethodPost, path, "", nil)
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
	resp, err := c.Do(http.MethodDelete, path, "", nil)
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

	resp, err := c.Do(http.MethodPost, topicsPath, contentTypeJSON, send)
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
	resp, err := c.Do(http.MethodDelete, path, "", nil)
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

	path := fmt.Sprintf(topicRecordsPath, topicName, fromPartition, toOffset)

	if toOffset < 0 || fromPartition < 0 {
		return NewResourceError(http.StatusBadRequest, c.Config.Host+"/"+path, "DELETE", "offset and partition should be positive numbers")
	}

	resp, err := c.Do(http.MethodDelete, path, "", nil)
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
	resp, err := c.Do(http.MethodPut, path, contentTypeJSON, send)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

// Topic describes the data that the `GetTopic` returns.
type Topic struct {
	TopicName            string             `json:"topicName" header:"Name"`
	KeyType              string             `json:"keyType" header:"Key /,NULL"`        // maybe string-based enum?
	ValueType            string             `json:"valueType" header:"Value Type,NULL"` // maybe string-based enum?
	Partitions           int                `json:"partitions" header:"Part"`
	Replication          int                `json:"replication" header:"Repl"`
	IsControlTopic       bool               `json:"isControlTopic"`
	KeySchema            string             `json:"keySchema,omitempty"`
	ValueSchema          string             `json:"valueSchema,omitempty"`
	MessagesPerSecond    int64              `json:"messagesPerSecond" header:"msg/sec"`
	TotalMessages        int64              `json:"totalMessages" header:"Total Msg"`
	Timestamp            int64              `json:"timestamp"`
	Config               []KV               `json:"config" header:"Configs,count"`
	ConsumersGroup       []ConsumersGroup   `json:"consumers"`
	MessagesPerPartition []PartitionMessage `json:"messagesPerPartition"`
	IsMarkedForDeletion  bool               `json:"isMarkedForDeletion" header:"Marked Del"`
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
	resp, respErr := c.Do(http.MethodGet, path, "", nil)
	if respErr != nil {
		err = respErr
		return
	}

	err = c.ReadJSON(resp, &topic)
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

	resp, err := c.Do(http.MethodPost, processorsPath, contentTypeJSON, send)
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
		ID              string `json:"id"` // header:"ID,text"`
		Name            string `json:"name" header:"Name"`
		DeploymentState string `json:"deploymentState" header:"State"`
		Runners         int    `json:"runners" header:"Runners"`
		User            string `json:"user" header:"Created By"`
		StartTimestamp  int64  `json:"startTs" header:"Started at,timestamp(ms|02 Jan 2006 15:04)"`
		StopTimestamp   int64  `json:"stopTs,omitempty"` // header:"Stopped,timestamp(ms|02 Jan 2006 15:04),No"`
		Uptime          int64  `json:"uptime" header:"Up time,unixduration"`

		Namespace   string `json:"namespace" header:"Namespace"`
		ClusterName string `json:"clusterName" header:"Cluster"`

		SQL string `json:"sql"` // header:"SQL"`

		TopicValueDecoder string `json:"topicValueDecoder"` // header:"Topic Decoder"`
		Pipeline          string `json:"pipeline"`          // header:"Pipeline"`

		ToTopics               []string `json:"toTopics,omitempty"` // header:"To Topics"`
		FromTopics             []string `json:"fromTopics,omitempty"`
		LastActionMessage      string   `json:"lastActionMsg,omitempty"`      // header:"Last Action"`
		DeploymentErrorMessage string   `json:"deploymentErrorMsg,omitempty"` // header:"Depl Error"`

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

	resp, err := c.Do(http.MethodGet, processorsPath, "", nil)
	if err != nil {
		return res, err
	}

	if err = c.ReadJSON(resp, &res); err != nil {
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
	resp, err := c.Do(http.MethodPut, path, "", nil)
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
	resp, err := c.Do(http.MethodPut, path, "", nil)
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
	resp, err := c.Do(http.MethodPut, path, "", nil)
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
	resp, err := c.Do(http.MethodDelete, path, "", nil)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

//
// Connector API
// https://docs.confluent.io/current/connect/devguide.html
// https://docs.confluent.io/current/connect/restapi.html
// http://lenses.stream/dev/lenses-apis/rest-api/index.html#connector-api
//

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
	// Name of the created (or received) connector.
	ClusterName string `json:"clusterName,omitempty" header:"Cluster"` // internal use only, not set by response.
	Name        string `json:"name" header:"Name"`

	// Config parameters for the connector
	Config ConnectorConfig `json:"config,omitempty" header:"Configs,count"`
	// Tasks is the list of active tasks generated by the connector.
	Tasks []ConnectorTaskReadOnly `json:"tasks,omitempty" header:"Tasks,count"`
}

const connectorsPath = "api/proxy-connect/%s/connectors"

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
	resp, respErr := c.Do(http.MethodGet, path, contentTypeJSON, nil)
	if respErr != nil {
		err = respErr
		return
	}

	err = c.ReadJSON(resp, &names)
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
// config (map) – Config parameters for the connector. All values should be strings.
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
	resp, respErr := c.Do(http.MethodPost, path, contentTypeJSON, send)
	if respErr != nil {
		err = respErr
		return
	}

	// re-use of the connector payload.
	err = c.ReadJSON(resp, &connector)
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
	resp, respErr := c.Do(http.MethodPut, path, contentTypeJSON, send)
	if respErr != nil {
		err = respErr
		return
	}

	err = c.ReadJSON(resp, &connector)
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
	resp, respErr := c.Do(http.MethodGet, path, contentTypeJSON, nil)
	if respErr != nil {
		err = respErr
		return
	}

	err = c.ReadJSON(resp, &connector)
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
	resp, respErr := c.Do(http.MethodGet, path, contentTypeJSON, nil)
	if respErr != nil {
		err = respErr
		return
	}

	err = c.ReadJSON(resp, &cfg)
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
		Name      string                        `json:"name" header:"Name"`
		Connector ConnectorStatusConnectorField `json:"connector" header:"inline"`
		Tasks     []ConnectorStatusTask         `json:"tasks,omitempty" header:"Tasks,count"`
	}

	// ConnectorStatusConnectorField describes a connector's status,
	// see `ConnectorStatus`.
	ConnectorStatusConnectorField struct {
		State    string `json:"state" header:"State"`      // i.e RUNNING
		WorkerID string `json:"worker_id" header:"Worker"` // i.e fakehost:8083
	}

	// ConnectorStatusTask describes a connector task's status,
	// see `ConnectorStatus`.
	ConnectorStatusTask struct {
		ID       int    `json:"id" header:"ID,text"`                  // i.e 1
		State    string `json:"state" header:"State"`                 // i.e FAILED
		WorkerID string `json:"worker_id" header:"Worker"`            // i.e fakehost:8083
		Trace    string `json:"trace,omitempty" header:"Trace,empty"` // i.e org.apache.kafka.common.errors.RecordTooLargeException\n
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
	resp, respErr := c.Do(http.MethodGet, path, "", nil)
	if respErr != nil {
		err = respErr
		return
	}

	err = c.ReadJSON(resp, &cs)
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
	resp, err := c.Do(http.MethodPut, path, "", nil) // the success status is 202 Accepted.
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
	resp, err := c.Do(http.MethodPut, path, "", nil)
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
	resp, err := c.Do(http.MethodPost, path, "", nil)
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
	resp, err := c.Do(http.MethodDelete, path, "", nil)
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
	resp, respErr := c.Do(http.MethodGet, path, "", nil)
	if respErr != nil {
		err = respErr
		return
	}

	err = c.ReadJSON(resp, &m)
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
	resp, respErr := c.Do(http.MethodGet, path, "", nil)
	if respErr != nil {
		err = respErr
		return
	}

	err = c.ReadJSON(resp, &cst)
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
	resp, err := c.Do(http.MethodPost, path, "", nil)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

// ConnectorPlugin describes the entry data of the list that are being received from the `GetConnectorPlugins`.
type ConnectorPlugin struct {
	// Class is the connector class name.
	Class string `json:"class" header:"Class"`

	Type string `json:"type" header:"Type"`

	Version string `json:"version" header:"Version"`
}

const pluginsPath = "api/proxy-connect/%s/connector-plugins"

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
	resp, respErr := c.Do(http.MethodGet, path, "", nil)
	if respErr != nil {
		err = respErr
		return
	}

	err = c.ReadJSON(resp, &cp)
	return
}

//
// Schemas (and Subjects) API
// https://docs.confluent.io/current/schema-registry/docs/api.html
//

const schemaAPIVersion = "v1"
const contentTypeSchemaJSON = "application/vnd.schemaregistry." + schemaAPIVersion + "+json"

const subjectsPath = "api/proxy-sr/subjects"

// GetSubjects returns a list of the available subjects(schemas).
// https://docs.confluent.io/current/schema-registry/docs/api.html#subjects
func (c *Client) GetSubjects() (subjects []string, err error) {
	// # List all available subjects
	// GET /api/proxy-sr/subjects
	resp, respErr := c.Do(http.MethodGet, subjectsPath, "", nil, schemaAPIOption)
	if respErr != nil {
		err = respErr
		return
	}

	err = c.ReadJSON(resp, &subjects)
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
	resp, respErr := c.Do(http.MethodGet, path, "", nil, schemaAPIOption)
	if respErr != nil {
		err = respErr
		return
	}

	err = c.ReadJSON(resp, &versions)
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
	resp, respErr := c.Do(http.MethodDelete, path, "", nil, schemaAPIOption)
	if respErr != nil {
		err = respErr
		return
	}

	err = c.ReadJSON(resp, &versions)
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
	resp, err := c.Do(http.MethodGet, path, "", nil, schemaAPIOption)
	if err != nil {
		return "", err
	}

	var res schemaOnlyJSON
	if err = c.ReadJSON(resp, &res); err != nil {
		return "", err
	}

	return res.Schema, nil
}

// Schema describes a schema, look `GetSchema` for more.
type Schema struct {
	ID int `json:"id,omitempty" yaml:"ID,omitempty" header:"ID,text"`
	// Name is the name of the schema is registered under.
	Name string `json:"subject,omitempty" yaml:"Name" header:"Name"` // Name is the "subject" argument in client-code, this structure is being used on CLI for yaml-file based loading.
	// Version of the returned schema.
	Version int `json:"version" header:"Version"`
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
	resp, respErr := c.Do(http.MethodGet, path, "", nil, schemaAPIOption)
	if respErr != nil {
		err = respErr
		return
	}

	err = c.ReadJSON(resp, &s)
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
	resp, err := c.Do(http.MethodPost, path, contentTypeSchemaJSON, send, schemaAPIOption)
	if err != nil {
		return 0, err
	}

	var res idOnlyJSON
	err = c.ReadJSON(resp, &res)
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
	resp, err := c.Do(http.MethodDelete, path, contentTypeSchemaJSON, nil, schemaAPIOption)
	if err != nil {
		return 0, err
	}

	var res int
	err = c.ReadJSON(resp, &res)

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

const compatibilityLevelPath = "api/proxy-sr/config"

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
	resp, err := c.Do(http.MethodPut, compatibilityLevelPath, contentTypeSchemaJSON, send, schemaAPIOption)
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
	resp, respErr := c.Do(http.MethodGet, compatibilityLevelPath, "", nil, schemaAPIOption)
	if respErr != nil {
		err = respErr
		return
	}

	var levelReq compatibilityOnlyJSON
	err = c.ReadJSON(resp, &levelReq)
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
	resp, err := c.Do(http.MethodPut, path, contentTypeSchemaJSON, send, schemaAPIOption)
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
	resp, respErr := c.Do(http.MethodGet, path, "", nil, schemaAPIOption)
	if respErr != nil {
		err = respErr
		return
	}

	var levelReq compatibilityOnlyJSON
	err = c.ReadJSON(resp, &levelReq)
	level = CompatibilityLevel(levelReq.Compatibility)

	return
}

//
// ACL API
// "ACL" stands for "Access Control Lists".
//

// ACLOperation is a string and it defines the valid operations for ACL.
//
// Based on:
// https://github.com/apache/kafka/blob/1.0/clients/src/test/java/org/apache/kafka/common/acl/AclOperationTest.java#L38
//
// Read through `ACLOperations` to learn what operation is valid for each of the available resource types.
type ACLOperation string

const (
	// OpUnknown is the kafka internal "UNKNOWN" ACL operation which is returned
	// if invalid operation passed.
	// ACLOpUnknown ACLOperation = "unknown"

	// ACLOperationAny is the "ANY" ACL operation.
	ACLOperationAny ACLOperation = "ANY"
	// ACLOperationAll is the "ALL" ACL operation.
	ACLOperationAll ACLOperation = "ALL"
	// ACLOperationRead is the "READ" ACL operation.
	ACLOperationRead ACLOperation = "READ"
	// ACLOperationWrite is the "WRITE" ACL operation.
	ACLOperationWrite ACLOperation = "WRITE"
	// ACLOperationCreate is the "CREATE" ACL operation.
	ACLOperationCreate ACLOperation = "CREATE"
	// ACLOperationDelete is the "DELETE" ACL operation.
	ACLOperationDelete ACLOperation = "DELETE"
	// ACLOperationAlter is the "ALTER" ACL operation.
	ACLOperationAlter ACLOperation = "ALTER"
	// ACLOperationDescribe is the "DESCRIBE" ACL operation.
	ACLOperationDescribe ACLOperation = "DESCRIBE"
	// ACLOperationClusterAction is the "CLUSTER_ACTION" ACL operation.
	ACLOperationClusterAction ACLOperation = "CLUSTER_ACTION"
	// ACLOperationDescribeConfigs is the "DESCRIBE_CONFIGS" ACL operation.
	ACLOperationDescribeConfigs ACLOperation = "DESCRIBE_CONFIGS"
	// ACLOperationAlterConfigs is the "ALTER_CONFIGS" ACL operation.
	ACLOperationAlterConfigs ACLOperation = "ALTER_CONFIGS"
	// ACLOperationIdempotentWrite is the "IDEMPOTENT_WRITE" ACL operation.
	ACLOperationIdempotentWrite ACLOperation = "IDEMPOTENT_WRITE"
)

// ACLResourceType is a string and it defines the valid resource types for ACL.
//
// Based on:
// https://github.com/apache/kafka/blob/1.0/clients/src/test/java/org/apache/kafka/common/resource/ResourceTypeTest.java#L38
type ACLResourceType string

const (
	// ACLResourceUnknown is the kafka internal "UNKNOWN" ACL resource type which is
	// returned if invalid resource type passed.
	// ACLResourceUnknown ACLResourceType = "unknown"

	// ACLResourceAny is the "ANY" ACL resource type.
	ACLResourceAny ACLResourceType = "ANY"
	// ACLResourceTopic is the "TOPIC" ACL resource type.
	ACLResourceTopic ACLResourceType = "TOPIC"
	// ACLResourceGroup is the "GROUP" ACL resource type.
	ACLResourceGroup ACLResourceType = "GROUP"
	// ACLResourceCluster is the "CLUSTER" ACL resource type.
	ACLResourceCluster ACLResourceType = "CLUSTER"
	// ACLResourceTransactionalID is the "TRANSACTIONAL_ID" ACL resource type.
	ACLResourceTransactionalID ACLResourceType = "TRANSACTIONAL_ID"
	// ACLResourceDelegationToken is the "DELEGATION_TOKEN" ACL resource type,
	// available only on kafka version 1.1+.
	ACLResourceDelegationToken ACLResourceType = "DELEGATION_TOKEN"
)

// ACLOperations is a map which contains the allowed ACL operations(values) per resource type(key).
//
// Based on:
// https://docs.confluent.io/current/kafka/authorization.html#acl-format
var ACLOperations = map[ACLResourceType][]ACLOperation{
	ACLResourceTopic: {
		ACLOperationAll,
		ACLOperationRead,
		ACLOperationWrite,
		ACLOperationDescribe,
		ACLOperationDescribeConfigs,
		ACLOperationAlterConfigs,
	},
	ACLResourceGroup: {
		ACLOperationAll,
		ACLOperationRead,
		ACLOperationDescribe,
		ACLOperationDelete,
	},
	ACLResourceCluster: {
		ACLOperationAll,
		ACLOperationCreate,
		ACLOperationClusterAction,
		ACLOperationDescribe,
		ACLOperationDescribeConfigs,
		ACLOperationAlter,
		ACLOperationAlterConfigs,
		ACLOperationIdempotentWrite,
	},
	ACLResourceTransactionalID: {
		ACLOperationAll,
		ACLOperationDescribe,
		ACLOperationWrite,
	},
	ACLResourceDelegationToken: {
		ACLOperationAll,
		ACLOperationDescribe,
	},
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
//
// Based on: https://github.com/apache/kafka/blob/1.0/core/src/main/scala/kafka/security/auth/PermissionType.scala
type ACLPermissionType string

const (
	// ACLPermissionAllow is the "Allow" ACL permission type.
	ACLPermissionAllow ACLPermissionType = "Allow"
	// ACLPermissionDeny is the "Deny" ACL permission type.
	ACLPermissionDeny ACLPermissionType = "Deny"
)

// ACL is the type which defines a single Apache Access Control List.
type ACL struct {
	ResourceName   string            `json:"resourceName" yaml:"ResourceName" header:"Name"`           // required.
	ResourceType   ACLResourceType   `json:"resourceType" yaml:"ResourceType" header:"Type"`           // required.
	Principal      string            `json:"principal" yaml:"Principal" header:"Principal"`            // required.
	PermissionType ACLPermissionType `json:"permissionType" yaml:"PermissionType" header:"Permission"` // required.
	Host           string            `json:"host" yaml:"Host" header:"Host"`                           // required.
	Operation      ACLOperation      `json:"operation" yaml:"Operation" header:"Operation"`            // required.
}

// Validate force validates the acl's resource type, permission type and operation.
// It returns an error if the operation is not valid for the resource type.
func (acl *ACL) Validate() error {
	if string(acl.Operation) == "*" {
		acl.Operation = ACLOperationAll
	}

	// upper the first letter here on the resourceType, permissionType and operation before any action,
	// although kafka internally accepts both lowercase and uppercase.
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

const aclPath = "api/acl"

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

	resp, err := c.Do(http.MethodPut, aclPath, contentTypeJSON, send)
	if err != nil {
		return err
	}

	// note: the status code errors are checked in the `do` on every request.
	return resp.Body.Close()
}

// GetACLs returns all the available Apache Kafka Access Control Lists.
func (c *Client) GetACLs() ([]ACL, error) {
	resp, err := c.Do(http.MethodGet, aclPath, "", nil)
	if err != nil {
		return nil, err
	}

	var acls []ACL
	err = c.ReadJSON(resp, &acls)
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

	resp, err := c.Do(http.MethodDelete, aclPath, contentTypeJSON, send)
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
		// Entityname is the Kafka client id for "CLIENT"
		// and "CLIENTS" and user name for "USER", "USER" and "USERCLIENT", the `QuotaEntityXXX`.
		EntityName string `json:"entityName" yaml:"EntityName" header:"Name"`
		// EntityType can be either `QuotaEntityClient`, `QuotaEntityClients`,
		// `QuotaEntityClientsDefault`, `QuotaEntityUser`, `QuotaEntityUsers`, `QuotaEntityUserClient`
		// or `QuotaEntityUsersDefault`.
		EnityType QuotaEntityType `json:"entityType" yaml:"EntityType" header:"Type"`
		// Child is optional and only present for entityType `QuotaEntityUserClient` and is the client id.
		Child string `json:"child,omitempty" yaml:"Child"` // header:"Child"`
		// Properties  is a map of the quota constraints, the `QuotaConfig`.
		Properties QuotaConfig `json:"properties" yaml:"Properties" header:"inline"`
		// URL is the url from this quota in Lenses.
		URL string `json:"url" yaml:"URL"`

		IsAuthorized bool `json:"isAuthorized" yaml:"IsAuthorized"`
	}

	// QuotaConfig is a typed struct which defines the
	// map of the quota constraints, producer_byte_rate, consumer_byte_rate and request_percentage.
	QuotaConfig struct {
		// header note:
		// if "number" and no default value, then it will add "0", we use the empty space between commas to tell that the default value is space.
		ProducerByteRate  string `json:"producer_byte_rate" yaml:"ProducerByteRate" header:"Produce/sec, ,number"`
		ConsumerByteRate  string `json:"consumer_byte_rate" yaml:"ConsumerByteRate" header:"Consume/sec, ,number"`
		RequestPercentage string `json:"request_percentage" yaml:"RequestPercentage" header:"Request Percentage, ,number"`
	}
)

const quotasPath = "api/quotas"

// GetQuotas returns a list of all available quotas.
func (c *Client) GetQuotas() ([]Quota, error) {
	resp, err := c.Do(http.MethodGet, quotasPath, "", nil)
	if err != nil {
		return nil, err
	}

	var quotas []Quota
	err = c.ReadJSON(resp, &quotas)
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

	resp, err := c.Do(http.MethodPut, quotasPathAllUsers, contentTypeJSON, send)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

// DefaultQuotaConfigPropertiesToRemove is a set of hard-coded strings that the client will send on `DeleteQuotaXXX` functions.
// It contains the "producer_byte_rate", "consumer_byte_rate" and "request_percentage" as they're described at the `QuotaConfig` structure.
var DefaultQuotaConfigPropertiesToRemove = []string{"producer_byte_rate", "consumer_byte_rate", "request_percentage"}

func marshalQuotaConfigPropertiesToBeRemoved(propertiesToRemove []string) ([]byte, error) {
	for i, s := range propertiesToRemove {
		// if it's not empty but it contains an empty item, delete it.
		if s == "" {
			if len(propertiesToRemove) > i+1 {
				propertiesToRemove = append(propertiesToRemove[:i], propertiesToRemove[i+1:]...)
			} else {
				propertiesToRemove = []string{}
			}
		}
	}

	// it's empty, add its own.
	if len(propertiesToRemove) == 0 {
		propertiesToRemove = DefaultQuotaConfigPropertiesToRemove
	}

	return json.Marshal(propertiesToRemove)
}

// DeleteQuotaForAllUsers deletes the default for all users.
// Read more at: http://lenses.stream/using-lenses/user-guide/quotas.html.
//
// if "propertiesToRemove" is not passed or empty then the client will send all the available keys to be removed, see `DefaultQuotaConfigPropertiesToRemove` for more.
func (c *Client) DeleteQuotaForAllUsers(propertiesToRemove ...string) error {
	send, err := marshalQuotaConfigPropertiesToBeRemoved(propertiesToRemove)
	if err != nil {
		return err
	}

	resp, err := c.Do(http.MethodDelete, quotasPathAllUsers, contentTypeJSON, send)
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
	resp, err := c.Do(http.MethodPut, path, contentTypeJSON, send)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

// DeleteQuotaForUser deletes a quota for a user.
// if "propertiesToRemove" is not passed or empty then the client will send all the available keys to be removed, see `DefaultQuotaConfigPropertiesToRemove` for more.
func (c *Client) DeleteQuotaForUser(user string, propertiesToRemove ...string) error {
	send, err := marshalQuotaConfigPropertiesToBeRemoved(propertiesToRemove)
	if err != nil {
		return err
	}

	path := fmt.Sprintf(quotasPathUser, user)
	resp, err := c.Do(http.MethodDelete, path, contentTypeJSON, send)
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
	resp, err := c.Do(http.MethodPut, path, contentTypeJSON, send)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

// DeleteQuotaForUserAllClients deletes for all client ids for a user.
//
// if "propertiesToRemove" is not passed or empty then the client will send all the available keys to be removed, see `DefaultQuotaConfigPropertiesToRemove` for more.
func (c *Client) DeleteQuotaForUserAllClients(user string, propertiesToRemove ...string) error {
	send, err := marshalQuotaConfigPropertiesToBeRemoved(propertiesToRemove)
	if err != nil {
		return err
	}

	path := fmt.Sprintf(quotasPathUserAllClients, user)
	resp, err := c.Do(http.MethodDelete, path, contentTypeJSON, send)
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
	resp, err := c.Do(http.MethodPut, path, contentTypeJSON, send)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

// DeleteQuotaForUserClient deletes the quota for a user/client pair.
//
// if "propertiesToRemove" is not passed or empty then the client will send all the available keys to be removed, see `DefaultQuotaConfigPropertiesToRemove` for more.
func (c *Client) DeleteQuotaForUserClient(user, clientID string, propertiesToRemove ...string) error {
	send, err := marshalQuotaConfigPropertiesToBeRemoved(propertiesToRemove)
	if err != nil {
		return err
	}

	path := fmt.Sprintf(quotasPathUserClient, user, clientID)
	resp, err := c.Do(http.MethodDelete, path, contentTypeJSON, send)
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

	resp, err := c.Do(http.MethodPut, quotasPathAllClients, contentTypeJSON, send)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

// DeleteQuotaForAllClients deletes the default quota for all clients.
//
// if "propertiesToRemove" is not passed or empty then the client will send all the available keys to be removed, see `DefaultQuotaConfigPropertiesToRemove` for more.
func (c *Client) DeleteQuotaForAllClients(propertiesToRemove ...string) error {
	send, err := marshalQuotaConfigPropertiesToBeRemoved(propertiesToRemove)
	if err != nil {
		return err
	}

	resp, err := c.Do(http.MethodDelete, quotasPathAllClients, contentTypeJSON, send)
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
	resp, err := c.Do(http.MethodPut, path, contentTypeJSON, send)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

// DeleteQuotaForClient deletes quotas for a client id.
//
// if "propertiesToRemove" is not passed or empty then the client will send all the available keys to be removed, see `DefaultQuotaConfigPropertiesToRemove` for more.
func (c *Client) DeleteQuotaForClient(clientID string, propertiesToRemove ...string) error {
	send, err := marshalQuotaConfigPropertiesToBeRemoved(propertiesToRemove)
	if err != nil {
		return err
	}

	path := fmt.Sprintf(quotasPathClient, clientID)
	resp, err := c.Do(http.MethodDelete, path, contentTypeJSON, send)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

// Alert API

type (
	// AlertSetting describes the type of list entry of the `GetAlertSetting` and `CreateOrUpdateAlertSettingCondition`.
	AlertSetting struct {
		ID                int               `json:"id" header:"ID,text"`
		Description       string            `json:"description" header:"Desc"`
		Category          string            `json:"category" header:"Category"`
		Enabled           bool              `json:"enabled" header:"Enabled"`
		IsAvailable       bool              `json:"isAvailable" header:"Available"`
		Docs              string            `json:"docs,omitempty" header:"Docs"`
		ConditionTemplate string            `json:"conditionTemplate,omitempty" header:"Cond Tmpl"`
		ConditionRegex    string            `json:"conditionRegex,omitempty" header:"Cond Regex"`
		Conditions        map[string]string `json:"conditions,omitempty" header:"Conds"`
	}

	// AlertSettings describes the type of list entry of the `GetAlertSettings`.
	AlertSettings struct {
		Categories AlertSettingsCategoryMap `json:"categories" header:"inline"`
	}

	// AlertSettingsCategoryMap describes the type of `AlertSetting`'s Categories.
	AlertSettingsCategoryMap struct {
		Infrastructure []AlertSetting `json:"Infrastructure" header:"Infrastructure"`
		Consumers      []AlertSetting `json:"Consumers" header:"Consumers"`
	}
)

const (
	alertsPath                 = "api/alerts"
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
	resp, err := c.Do(http.MethodGet, alertSettingsPath, "", nil)
	if err != nil {
		return AlertSettings{}, err
	}

	var settings AlertSettings
	err = c.ReadJSON(resp, &settings)
	return settings, err
}

// GetAlertSetting returns a specific alert setting based on its "id".
func (c *Client) GetAlertSetting(id int) (setting AlertSetting, err error) {
	path := fmt.Sprintf(alertSettingPath, id)
	resp, respErr := c.Do(http.MethodGet, path, "", nil)
	if respErr != nil {
		err = respErr
		return
	}

	err = c.ReadJSON(resp, &setting)
	return
}

// EnableAlertSetting enables a specific alert setting based on its "id".
func (c *Client) EnableAlertSetting(id int) error {
	path := fmt.Sprintf(alertSettingPath, id)
	resp, err := c.Do(http.MethodPut, path, "", nil)
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
	resp, err := c.Do(http.MethodGet, path, "", nil)
	if err != nil {
		return nil, err
	}

	var conds AlertSettingConditions
	if err = c.ReadJSON(resp, &conds); err != nil {
		return nil, err
	}
	return conds, nil
}

type (
	// Alert is the request payload that is used to register an Alert via `RegisterAlert` and the response that client retrieves from the `GetAlerts`.
	Alert struct {
		// AlertID  is a unique identifier for the setting corresponding to this alert. See the available ids via `GetAlertSettings`.
		AlertID int `json:"alertId" yaml:"AlertID" header:"ID,text"`

		// Labels field is a list of key-value pairs. It must contain a non empty `Severity` value.
		Labels AlertLabels `json:"labels" yaml:"Labels" header:"inline"`
		// Annotations is a list of key-value pairs. It contains the summary, source, and docs fields.
		Annotations AlertAnnotations `json:"annotations" yaml:"Annotations"` // header:"inline"`
		// GeneratorURL is a unique URL identifying the creator of this alert.
		// It matches AlertManager requirements for providing this field.
		GeneratorURL string `json:"generatorURL" yaml:"GeneratorURL"` // header:"Gen URL"`

		// StartsAt is the time as string, in ISO format, for when the alert starts
		StartsAt string `json:"startsAt" yaml:"StartsAt" header:"Start,date"`
		// EndsAt is the time as string the alert ended at.
		EndsAt string `json:"endsAt" yaml:"EndsAt" header:"End,date"`
	}

	// AlertLabels labels for the `Alert`, at least Severity should be filled.
	AlertLabels struct {
		Category string `json:"category,omitempty" yaml:"Category,omitempty" header:"Category"`
		Severity string `json:"severity" yaml:"Severity,omitempty" header:"Severity"`
		Instance string `json:"instance,omitempty" yaml:"Instance,omitempty" header:"Instance"`
	}

	// AlertAnnotations annotations for the `Alert`, at least Summary should be filled.
	AlertAnnotations struct {
		Summary string `json:"summary" yaml:"Summary" header:"Summary"`
		Source  string `json:"source,omitempty" yaml:"Source,omitempty" header:"Source,empty"`
		Docs    string `json:"docs,omitempty" yaml:"Docs,omitempty" header:"Docs,empty"`
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

	resp, err := c.Do(http.MethodPost, alertsPath, contentTypeJSON, send)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

// GetAlerts returns the registered alerts.
func (c *Client) GetAlerts() (alerts []Alert, err error) {
	resp, respErr := c.Do(http.MethodGet, alertsPath, "", nil)
	if respErr != nil {
		err = respErr
		return
	}

	err = c.ReadJSON(resp, &alerts)
	return
}

// CreateOrUpdateAlertSettingCondition sets a condition(expression text) for a specific alert setting.
func (c *Client) CreateOrUpdateAlertSettingCondition(alertSettingID int, condition string) error {
	path := fmt.Sprintf(alertSettingConditionsPath, alertSettingID)
	resp, err := c.Do(http.MethodPost, path, "text/plain", []byte(condition))
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

// DeleteAlertSettingCondition deletes a condition from an alert setting.
func (c *Client) DeleteAlertSettingCondition(alertSettingID int, conditionUUID string) error {
	path := fmt.Sprintf(alertSettingConditionPath, alertSettingID, conditionUUID)
	resp, err := c.Do(http.MethodDelete, path, "", nil)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

const (
	alertsPathSSE = "api/sse/alerts"
)

// AlertHandler is the type of func that can be registered to receive alerts via the `GetAlertsLive`.
type AlertHandler func(Alert) error

// GetAlertsLive receives alert notifications in real-time from the server via a Send Server Event endpoint.
func (c *Client) GetAlertsLive(handler AlertHandler) error {
	resp, err := c.Do(http.MethodGet, alertsPathSSE, contentTypeJSON, nil, func(r *http.Request) error {
		r.Header.Add(acceptHeaderKey, "application/json, text/event-stream")
		return nil
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

		message := line[shiftN:] // we need everything after the 'data:'.

		if len(message) < 2 {
			continue // do NOT stop here, let the connection active.
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

const processorsLogsPathSSE = "api/sse/k8/logs/%s/%s/%s"

type processorLog struct {
	Timestamp string `json:"@timestamp" Header:"Timestamp,date"`
	Version   int    `json:"@version" Header:"Version"`
	Message   string `json:"message" Header:"Message"`
	// logger_name
	// thread_name
	Level string `json:"level"`
	// level_value
}

const defaultProcessorsLogsFollowLines = 100

// GetProcessorsLogs retrieves the LSQL processor logs if in kubernetes mode.
func (c *Client) GetProcessorsLogs(clusterName, ns, podName string, follow bool, lines int, handler func(level string, log string) error) error {
	if mode, _ := c.GetExecutionMode(); mode != ExecutionModeKubernetes {
		return fmt.Errorf("unable to retrieve logs, execution mode is not KUBERNETES")
	}

	path := fmt.Sprintf(processorsLogsPathSSE, clusterName, ns, podName)
	if follow {
		if lines <= 0 {
			lines = defaultProcessorsLogsFollowLines
		}

		path += "?follow=true&lines=" + fmt.Sprintf("%d", lines)
	}

	resp, err := c.Do(http.MethodGet, path, contentTypeJSON, nil, func(r *http.Request) error {
		r.Header.Add(acceptHeaderKey, "application/json, text/event-stream")
		return nil
	})
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
			continue
		}

		message := line[shiftN:]

		if len(message) < 2 {
			continue
		}

		// it can be a json object or a pure string log (but always after data:, i.e data:======> Log level set to INFO).
		logEntry := processorLog{}
		if message[0] == '{' {
			if err = json.Unmarshal(message, &logEntry); err == nil {
				t, err := time.Parse(time.RFC3339, logEntry.Timestamp)
				if err == nil {
					logEntry.Timestamp = t.Format("2006-01-02 15:04:05")
				}

				// colorized by the caller.
				if err = handler(logEntry.Level, fmt.Sprintf("%s %s", logEntry.Timestamp, logEntry.Message)); err != nil {
					return err
				}

			} else {
				// for any case.
				handler("info", string(message))
			}

			continue
		}

		// it contains the log level itself.
		handler("", string(message))
	}
}

//
// Dynamic Broker Configurations API
//

// BrokerConfig describes the kafka broker's configurations.
type BrokerConfig struct {
	LogCleanerThreads int    `json:"log.cleaner.threads" yaml:"LogCleanerThreads" header:"Log Cleaner Threads"`
	CompressionType   string `json:"compression.type" yaml:"CompressionType" header:"Compression Type"`
	AdvertisedPort    int    `json:"advertised.port" yaml:"AdvertisedPort" header:"Advertised Port"`
}

const (
	brokersConfigsPath = "api/configs/brokers"
	brokerConfigsPath  = brokersConfigsPath + "/%d"
)

// GetDynamicClusterConfigs returns the dynamic updated configurations for a kafka cluster.
// Retrieves only the ones added/updated dynamically.
func (c *Client) GetDynamicClusterConfigs() (configs BrokerConfig, err error) {
	resp, respErr := c.Do(http.MethodGet, brokersConfigsPath, "", nil)
	if respErr != nil {
		err = respErr
		return
	}

	err = c.ReadJSON(resp, &configs)
	return
}

// GetDynamicBrokerConfigs returns the dynamic updated configurations for a kafka broker.
// Retrieves only the ones added/updated dynamically.
func (c *Client) GetDynamicBrokerConfigs(brokerID int) (config BrokerConfig, err error) {
	path := fmt.Sprintf(brokerConfigsPath, brokerID)
	resp, respErr := c.Do(http.MethodGet, path, "", nil)
	if respErr != nil {
		err = respErr
		return
	}

	err = c.ReadJSON(resp, &config)
	return
}

// UpdateDynamicClusterConfigs adds or updates cluster configuration dynamically.
func (c *Client) UpdateDynamicClusterConfigs(toAddOrUpdate BrokerConfig) error {
	send, err := json.Marshal(toAddOrUpdate)
	if err != nil {
		return err
	}

	resp, err := c.Do(http.MethodPut, brokersConfigsPath, contentTypeJSON, send)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

// UpdateDynamicBrokerConfigs adds or updates broker configuration dynamically.
func (c *Client) UpdateDynamicBrokerConfigs(brokerID int, toAddOrUpdate BrokerConfig) error {
	send, err := json.Marshal(toAddOrUpdate)
	if err != nil {
		return err
	}

	path := fmt.Sprintf(brokerConfigsPath, brokerID)
	resp, err := c.Do(http.MethodPut, path, contentTypeJSON, send)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

// DeleteDynamicClusterConfigs deletes cluster configuration(s) dynamically.
// It reverts the configuration to its default value.
func (c *Client) DeleteDynamicClusterConfigs(configKeysToBeReseted ...string) error {
	send, err := json.Marshal(configKeysToBeReseted)
	if err != nil {
		return err
	}

	resp, err := c.Do(http.MethodDelete, brokersConfigsPath, contentTypeJSON, send)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

// DeleteDynamicBrokerConfigs deletes a configuration for a broker.
// Deleting a configuration dynamically reverts it to its default value.
func (c *Client) DeleteDynamicBrokerConfigs(brokerID int, configKeysToBeReseted ...string) error {
	send, err := json.Marshal(configKeysToBeReseted)
	if err != nil {
		return err
	}

	path := fmt.Sprintf(brokerConfigsPath, brokerID)
	resp, err := c.Do(http.MethodDelete, path, contentTypeJSON, send)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

//
// Audit API
//

// AuditEntryType the go type for audit entry types, see the `AuditEntry` structure for more.
type AuditEntryType string

// The available audit entry types.
// Available types: AuditEntryTopic, AuditEntryTopicData, AuditEntryQuotas, AuditEntryBrokerConfig,
// AuditEntryACL, AuditEntrySchema, AuditEntryProcessor, AuditEntryConnector.
const (
	AuditEntryTopic        AuditEntryType = "TOPIC"
	AuditEntryTopicData    AuditEntryType = "TOPIC_DATA"
	AuditEntryQuotas       AuditEntryType = "QUOTAS"
	AuditEntryBrokerConfig AuditEntryType = "BROKER_CONFIG"
	AuditEntryACL          AuditEntryType = "ACL"
	AuditEntrySchema       AuditEntryType = "SCHEMA"
	AuditEntryProcessor    AuditEntryType = "PROCESSOR"
	AuditEntryConnector    AuditEntryType = "CONNECTOR"
)

// AuditEntryChange the go type describer for the audit entry changes, see the `AuditEntry` structure for more.
type AuditEntryChange string

// The available audit entry changes.
// Available types: AuditEntryAdd, AuditEntryRemove, AuditEntryUpdate, AuditEntryInsert.
const (
	AuditEntryAdd    AuditEntryChange = "ADD"
	AuditEntryRemove AuditEntryChange = "REMOVE"
	AuditEntryUpdate AuditEntryChange = "UPDATE"
	AuditEntryInsert AuditEntryChange = "INSERT"
)

// AuditEntry describes a lenses Audit Entry, used for audit logs API.
type AuditEntry struct {
	Type      AuditEntryType    `json:"type" yaml:"Type" header:"Type"`
	Change    AuditEntryChange  `json:"change" yaml:"Change" header:"Change"`
	UserID    string            `json:"userId" yaml:"User" header:"User         "` /* make it a little bigger than expected, it looks slightly better for this field*/
	Timestamp int64             `json:"timestamp" yaml:"Timestamp" header:"Date,timestamp(ms|utc|02 Jan 2006 15:04)"`
	Content   map[string]string `json:"content" yaml:"Content" header:"Content"`
}

const auditPath = "api/audit"

// GetAuditEntries returns the last buffered audit entries.
//
// Retrives the last N audit entries created.
// See `GetAuditEntriesLive` for real-time notifications.
func (c *Client) GetAuditEntries() (entries []AuditEntry, err error) {
	resp, err := c.Do(http.MethodGet, auditPath, "", nil)
	if err != nil {
		return nil, nil
	}

	err = c.ReadJSON(resp, &entries)
	return
}

// AuditEntryHandler is the type of the function, the listener which is
// the input parameter of the `GetAuditEntriesLive` API call.
type AuditEntryHandler func(AuditEntry) error

const auditPathSSE = "api/sse/audit"

// GetAuditEntriesLive returns the live audit notifications, see `GetAuditEntries` too.
func (c *Client) GetAuditEntriesLive(handler AuditEntryHandler) error {
	if handler == nil {
		return errRequired("handler")
	}

	resp, err := c.Do(http.MethodGet, auditPathSSE, contentTypeJSON, nil, func(r *http.Request) error {
		r.Header.Add(acceptHeaderKey, "application/json, text/event-stream")
		return nil
	})
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

		message := line[shiftN:] // we need everything after the 'data:'.

		if len(message) < 2 {
			continue // do NOT stop here, let the connection active.
		}

		entry := AuditEntry{}

		if err = json.Unmarshal(message, &entry); err != nil {
			// exit on first error here as well.
			return err
		}

		if err = handler(entry); err != nil {
			return err // stop on first error by the caller.
		}
	}
}

//
// Logs API
//

// LogLine represents the return value(s) of the `GetLogsInfo` and `GetLogsMetrics` calls.
type LogLine struct {
	Level      string `json:"level" header:"Level"`
	Thread     string `json:"thread"`
	Logger     string `json:"logger"`
	Message    string `json:"message" header:"Message"`
	Stacktrace string `json:"Stacktrace"`
	Timestmap  int64  `json:"Timestamp"`
	Time       string `json:"time" header:"Time"`
}

const (
	logsPath        = "api/logs"
	logsInfoPath    = logsPath + "/INFO"
	logsMetricsPath = logsPath + "/METRICS"
)

// GetLogsInfo returns the latest (512) INFO log lines.
func (c *Client) GetLogsInfo() ([]LogLine, error) {
	resp, err := c.Do(http.MethodGet, logsInfoPath, "", nil)
	if err != nil {
		return nil, err
	}
	var logs []LogLine
	err = c.ReadJSON(resp, &logs)
	return logs, err
}

// GetLogsMetrics returns the latest (512) METRICS log lines.
func (c *Client) GetLogsMetrics() ([]LogLine, error) {
	resp, err := c.Do(http.MethodGet, logsMetricsPath, "", nil)
	if err != nil {
		return nil, err
	}
	var logs []LogLine
	err = c.ReadJSON(resp, &logs)
	return logs, err
}

//
// User Profile API
//

// UserProfile contains all the user-specific favourites, only kafka related info.
type UserProfile struct {
	Topics       []string `json:"topics" header:"Topics"`
	Schemas      []string `json:"schemas" header:"Schemas"`
	Transformers []string `json:"transformers" header:"Transformers"`
}

const (
	userProfilePath         = "api/user/profile"
	userProfilePropertyPath = userProfilePath + "/%s/%s"
)

// GetUserProfile returns the user-specific favourites.
func (c *Client) GetUserProfile() (UserProfile, error) {
	var profile UserProfile

	resp, err := c.Do(http.MethodGet, userProfilePath, "", nil)
	if err != nil {
		return profile, err
	}

	err = c.ReadJSON(resp, &profile)
	return profile, err
}

// CreateUserProfilePropertyValue adds a "value" to the user profile "property" entries.
func (c *Client) CreateUserProfilePropertyValue(property, value string) error {
	path := fmt.Sprintf(userProfilePropertyPath, property, value)
	resp, err := c.Do(http.MethodPut, path, "", nil)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

// DeleteUserProfilePropertyValue removes the "value" from the user profile "property" entries.
func (c *Client) DeleteUserProfilePropertyValue(property, value string) error {
	path := fmt.Sprintf(userProfilePropertyPath, property, value)
	resp, err := c.Do(http.MethodDelete, path, "", nil)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

//
// Static API
//

const (
	staticPath                    = "api/static"
	staticSupportedConnectorsPath = staticPath + "/supported-connectors"
)

// ConnectorInfoUI describes a supported Kafka Connector, result type of the `GetSupportedConnectors` call.
type ConnectorInfoUI struct {
	Class       string `json:"class"` // header:"Class"`
	Name        string `json:"name" header:"Name"`
	Type        string `json:"type" header:"Type"`
	Version     string `json:"version" header:"Version"`
	Author      string `json:"author,omitempty" header:"Author"`
	Description string `json:"description,omitempty" header:"Desc"`
	Docs        string `json:"docs,omitempty"` // header:"Docs"`
	UIEnabled   bool   `json:"uiEnabled" header:"UI Enabled"`
}

// GetSupportedConnectors returns the list of the supported Kafka Connectors.
func (c *Client) GetSupportedConnectors() ([]ConnectorInfoUI, error) {
	resp, err := c.Do(http.MethodGet, staticSupportedConnectorsPath, "", nil)
	if err != nil {
		return nil, err
	}

	var connectorsInfo []ConnectorInfoUI
	err = c.ReadJSON(resp, &connectorsInfo)
	return connectorsInfo, err
}
