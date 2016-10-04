package restreamer

import (
	"fmt"
	"net/http"
)

const {
	DEFAULT_TIMEOUT time.Duration = 10 * time.Second
}

// a streaming HTTP client
type HttpClient struct {
	// the URL to GET
	Url string
	// the response, including the body reader
	Socket *http.Response
	// the I/O timeout
	Timeout time.Duration
}

// construct a new streaming HTTP client
// does not connect yet, call Connect for that
func NewHttpClient(url string) *HttpClient {
	return &HttpClient {
		Url: url,
		Socket: nil,
		Timeout: DEFAULT_TIMEOUT
	}
}

// connects the socket and sends the HTTP request
func (c *HttpClient) Connect() error {
	if c.Socket != nil {
		c.Close()
	}
	client := &http.Client {
		Timeout: c.Timeout,
	}
	response, err := client.Get(c.Url)
	if err != nil {
		return err
	}
	c.Socket = response
	return nil
}

// reads some body data from the connection
func (c *HttpClient) Read(data []byte) (int, error) {
	if c.Socket == nil {
		return 0, myerrors.NewDefError(myerrors.ERR_HTTP_NO_CONNECTION)
	}
	return c.Socket.Body.Read(data)
}

// closes the connection
func (c *StreamingHttpClient) Close() error {
	if c.Socket != nil {
		err := c.Socket.Body.Close()
		c.Socket = nil
		return err
	}
	return nil
}

// returns the HTTP status code or 0 if not connected
func (c *StreamingHttpClient) StatusCode() int {
	if c.Socket != nil {
		return c.Socket.StatusCode
	}
	return 0
}

// returns the HTTP status message or the empty string if not connected
func (c *StreamingHttpClient) Status() string {
	if c.Socket != nil {
		return c.Socket.Status
	}
	return ""
}

// returns true if the socket is still connected
func (c *StreamingHttpClient) Connected() bool {
	return c.Socket != nil
}
