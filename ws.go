package lenses

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kataras/golog"
	uuid "github.com/satori/go.uuid"
)

// RequestType is the corresponding action/message type for the request sent to the back-end server.
type RequestType string

const (
	// SubscribeRequest is the "SUBSCRIBE" action type sent to the back-end server.
	SubscribeRequest RequestType = "SUBSCRIBE"
	// UnsubscribeRequest is the "UNSUBSCRIBE" action type sent to the back-end server.
	UnsubscribeRequest RequestType = "UNSUBSCRIBE"
	// PublishRequest is the "PUBLISH" action type sent to the back-end server.
	PublishRequest RequestType = "PUBLISH"
	// CommitRequest is the "COMMIT" action type sent to the back-end server.
	CommitRequest RequestType = "COMMIT"
	// LoginRequest is the "LOGIN" action type sent to the back-end server.
	LoginRequest RequestType = "LOGIN"
)

// ResponseType is the corresponding message type for the response came from the back-end server to the client.
type ResponseType string

const (
	// WildcardResponse is a custom type only for the go library
	// which can be passed to the `On` event in order to catch all the incoming messages and fire the corresponding callback response handler.
	WildcardResponse ResponseType = "*"
	// ErrorResponse is the "ERROR" receive message type.
	ErrorResponse ResponseType = "ERROR"
	// InvalidRequestResponse is the "INVALIDREQUEST" receive message type.
	InvalidRequestResponse ResponseType = "INVALIDREQUEST"
	// KafkaMessageResponse is the "KAFKAMSG" receive message type.
	KafkaMessageResponse ResponseType = "KAFKAMSG"
	// HeartbeatResponse is the "HEARTBEAT" receive message type.
	HeartbeatResponse ResponseType = "HEARTBEAT"
	// SuccessResponse is the "SUCCESS" receive message type.
	SuccessResponse ResponseType = "SUCCESS"
)

type (
	// LiveRequest contains the necessary information that
	// the back-end websocket server waits to be sent by the websocket client.
	LiveRequest struct {
		// Type describes the action the back end will take in response to the request.
		// The available values are: "LOGIN", "SUBSCRIBE", "UNSUBSCRIBE",
		// "PUBLISH" and "COMMIT". The Go type is `RequestType`.
		Type RequestType `json:"type"`

		// CorrelationID is the unique identifier in order for the client to link the response
		// with the request made.
		CorrelationID int64 `json:"correlationId"`

		// Content contains the Json content of the actual request.
		// The content is strictly related to the type described shortly.
		Content string `json:"content"`

		// AuthToken is the unique token identifying the user making the request.
		// This token can only be obtained once the LOGIN request type has completed successfully.
		//
		// It's created automatically by the internal implementation,
		// on the `LivePublisher#Publish` which is used inside the `LiveListeners`.
		AuthToken string `json:"authToken"`
	}

	// LiveResponse contains the necessary information that
	// the websocket client expects to receive from the back-end websocket server.
	LiveResponse struct {
		// Type describes what response content the client has
		// received. Available values are: "ERROR",
		// "INVALIDREQUEST", "KAFKAMSG", "HEARTBEAT" and "SUCCESS". The Go type is `ResponseType`.
		Type ResponseType `json:"type"`

		// CorrelationID is the unique identifier the client has provided in |
		// the request associated with the response.
		CorrelationID int64 `json:"correlationId"`

		// Content contains the actual response content.
		// Each response type has its own content layout.
		Content json.RawMessage `json:"content"`
	}
)

type (
	// LiveConfiguration contains the contact information
	// about the websocket communication.
	// It contains the host(including the scheme),
	// the user and password credentials
	// and, optionally, the client id which is the kafka consumer group.
	//
	// See `OpenLiveConnection` for more.
	LiveConfiguration struct {
		Host     string `json:"host" yaml:"Host" toml:"Host"`
		User     string `json:"user" yaml:"User" toml:"User"`
		Password string `json:"password" yaml:"Password" toml:"Password"`
		ClientID string `json:"clientId,omitempty" yaml:"ClientID" toml:"ClientID"`
		Debug    bool   `json:"debug" yaml:"Debug" toml:"Debug"`
		// ws-specific settings, optionally.

		// HandshakeTimeout specifies the duration for the handshake to complete.
		HandshakeTimeout time.Duration
		// ReadBufferSize and WriteBufferSize specify I/O buffer sizes. If a buffer
		// size is zero, then a useful default size is used. The I/O buffer sizes
		// do not limit the size of the messages that can be sent or received.
		ReadBufferSize, WriteBufferSize int

		// TLSClientConfig specifies the TLS configuration to use with tls.Client.
		// If nil, the default configuration is used.
		TLSClientConfig *tls.Config
	}

	// LiveConnection is the websocket connection.
	LiveConnection struct {
		conn   *websocket.Conn
		config LiveConfiguration

		receiveStop chan struct{}
		closed      uint32

		authToken string // generated by the login and `OnSuccess` internal listener.
		endpoint  string // generated by the config's host and the client id.

		listeners map[ResponseType][]LiveListener
		mu        sync.RWMutex

		errors chan error // error comes from reader.
	}
)

// OpenLiveConnection starts the websocket communication
// and returns the client connection for further operations.
// An error will be returned if login failed.
//
// The `Err` function is used to report any
// reader's error, the reader operates on its own go routine.
//
// The connection starts reading immediately, the implementation is subscribed to the `Success` message
// to validate the login.
//
// Usage:
// c, err := lenses.OpenLiveConnection(lenses.LiveConfiguration{
//    [...]
// })
//
// c.On(lenses.KafkaMessageResponse, func(pub lenses.LivePublisher, response lenses.LiveResponse) error {
//    [...]
// })
//
// c.On(lenses.WildcardResponse, func(pub lenses.LivePublisher, response lenses.LiveResponse) error {
//    [...catch all messages]
// })
//
// c.OnSuccess(func(cub lenses.LivePublisher, response lenses.LiveResponse) error{
//    pub.Publish(lenses.SubscribeRequest, 2, `{"sqls": ["SELECT * FROM reddit_posts LIMIT 3"]}`)
// }) also OnKafkaMessage, OnError, OnHeartbeat, OnInvalidRequest.
//
// If at least one listener returned an error then the communication is terminated.
func OpenLiveConnection(config LiveConfiguration) (*LiveConnection, error) {
	if config.Debug {
		golog.SetLevel("debug")
	}

	if config.ClientID == "" {
		config.ClientID = uuid.Must(uuid.NewV4()).String()
	}

	if config.HandshakeTimeout == 0 {
		config.HandshakeTimeout = 45 * time.Second
	}

	config.Host = strings.Replace(config.Host, "https://", "wss://", 1)
	config.Host = strings.Replace(config.Host, "http://", "ws://", 1)

	c := &LiveConnection{
		config:      config,
		endpoint:    fmt.Sprintf("%s/api/kafka/ws/%s", config.Host, config.ClientID),
		receiveStop: make(chan struct{}),
		listeners:   make(map[ResponseType][]LiveListener),
		errors:      make(chan error),
	}

	return c, c.start()
}

func (c *LiveConnection) start() error {
	// first connect, handshake with the websocket server for upgradation.
	dialer := websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: c.config.HandshakeTimeout,
		ReadBufferSize:   c.config.ReadBufferSize,
		WriteBufferSize:  c.config.WriteBufferSize,
	}

	conn, _, err := dialer.Dial(c.endpoint, nil)

	if err != nil {
		err = fmt.Errorf("connect failure for '%s': %v", c.config.Host, err)
		golog.Debug(err)
		return err
	}
	// set the websocket connection.
	c.conn = conn

	// register the internal listener in order to check
	// for the login and set the generated authentication token
	// which should be used to send messages to the websocket server.
	c.OnSuccess(func(_ LivePublisher, resp LiveResponse) error {
		if resp.CorrelationID == 1 {
			// the login, otherwise it sends the topic name for example after the subscribe.
			err := json.Unmarshal(resp.Content, &c.authToken)
			if err == nil {
				golog.Debugf("login succeed, auth token: %s", c.authToken)
			}
			return err
		}
		return nil
	})

	go c.readLoop()

	// then login.
	if err := c.login(); err != nil {
		err = fmt.Errorf("login failure: %v", err)
		golog.Debug(err)
		return err
	}

	return nil
}

// Wait waits until interruptSignal fires, if it's nil then it waits for ever.
func (c *LiveConnection) Wait(interruptSignal <-chan os.Signal) error {
	select {
	case <-interruptSignal:
		return c.Close()
	}
}

// type userPayload struct {
// 	User     string `json:"user"`
// 	Password string `json:"password"`
// }

func makeLoginContent(user, password string) string {
	// u := userPayload{
	// 	User:     c.config.User,
	// 	Password: c.config.Password,
	// }
	// content, _ := json.Marshal(u)
	// return string(content)
	//
	// Or just make a simple json string,
	// the server will accept it as it's.
	return fmt.Sprintf(`{"user": "%s", "password": "%s"}`, user, password)
}

func (c *LiveConnection) login() error {
	req := LiveRequest{
		Type:          LoginRequest,
		CorrelationID: 1,
		Content:       makeLoginContent(c.config.User, c.config.Password),
	}

	return c.conn.WriteJSON(req)
}

// Err can be used to receive the errors coming from the communication,
// the listeners' errors are sending to that channel too.
func (c *LiveConnection) Err() <-chan error {
	return c.errors
}

func (c *LiveConnection) sendErr(err error) {
	golog.Debug(err)
	c.errors <- err
}

func (c *LiveConnection) readLoop() {
	defer c.Close() // close on any errors or loop break.
	for {
		select {
		case <-c.receiveStop:
			// golog.Debugf("stop receiving by signal")
			return
		default:
			resp := LiveResponse{}
			if err := c.conn.ReadJSON(&resp); err != nil {
				c.sendErr(fmt.Errorf("live: read json: %v", err))
				continue
			}

			golog.Debugf("read: %#+v", resp)

			// fire.
			c.mu.RLock()
			callbacks, ok := c.listeners[resp.Type]
			c.mu.RUnlock()

			if ok {
				for _, cb := range callbacks {
					if err := cb(c, resp); err != nil {
						// return err // break and exit the loop on first failure.
						c.sendErr(err) // don't break, just add the error.
					}
				}
			}
		}
	}
}

// --- Events handles incoming messages with style. ---

// LivePublisher is the interface which
// the `LiveConnection` implements, it is used on
// `LiveListeners` to send requests to the websocket server.
type LivePublisher interface {
	Publish(RequestType, int64, string) error
}

// Publish sends a `LiveRequest` based on the input arguments
// as JSON data to the websocket server.
func (c *LiveConnection) Publish(typ RequestType, correlationID int64, content string) error {
	req := LiveRequest{
		AuthToken:     c.authToken,
		Type:          typ,
		CorrelationID: correlationID,
		Content:       content,
	}

	golog.Debugf("publish: %#+v", req)

	return c.conn.WriteJSON(req)
}

// LiveListener is the declaration for the subscriber, the subscriber
// is just a callback which fiers whenever a websocket message
// with a particular `ResponseType` was sent by the websocket server.
//
// See `On` too.
type LiveListener func(LivePublisher, LiveResponse) error

// On adds a listener, a websocket message subscriber based on the given "typ" `ResponseType`.
// Use the `WildcardResponse` to subscribe to all message types.
func (c *LiveConnection) On(typ ResponseType, cb LiveListener) {
	if typ == WildcardResponse {
		c.OnError(cb)
		c.OnInvalidRequest(cb)
		c.OnKafkaMessage(cb)
		c.OnHeartbeat(cb)
		c.OnSuccess(cb)
		return
	}

	c.mu.Lock()
	c.listeners[typ] = append(c.listeners[typ], cb)
	c.mu.Unlock()
}

// OnError adds a listener, a websocket message subscriber based on the "ERROR" `ResponseType`.
func (c *LiveConnection) OnError(cb LiveListener) { c.On(ErrorResponse, cb) }

// OnInvalidRequest adds a listener, a websocket message subscriber based on the "INVALIDREQUEST" `ResponseType`.
func (c *LiveConnection) OnInvalidRequest(cb LiveListener) { c.On(InvalidRequestResponse, cb) }

// OnKafkaMessage adds a listener, a websocket message subscriber based on the "KAFKAMSG" `ResponseType`.
func (c *LiveConnection) OnKafkaMessage(cb LiveListener) { c.On(KafkaMessageResponse, cb) }

// OnHeartbeat adds a listener, a websocket message subscriber based on the "HEARTBEAT" `ResponseType`.
func (c *LiveConnection) OnHeartbeat(cb LiveListener) { c.On(HeartbeatResponse, cb) }

// OnSuccess adds a listener, a websocket message subscriber based on the "SUCCESS" `ResponseType`.
func (c *LiveConnection) OnSuccess(cb LiveListener) { c.On(SuccessResponse, cb) }

// Close closes the underline websocket connection
// and stops receiving any new message from the websocket server.
//
// If `Close` called more than once then it will return nil and nothing will happen.
func (c *LiveConnection) Close() error {
	golog.Debugf("terminating websocket connection...")
	// if we try to close a closed channel panic will occur,
	// in order to prevent it we've added an atomic checkpoint.
	if atomic.LoadUint32(&c.closed) > 0 {
		// means already closed.
		return nil
	}

	atomic.StoreUint32(&c.closed, 1)
	close(c.receiveStop) // stop receiving, see `readLoop`.
	return c.conn.Close()
}
