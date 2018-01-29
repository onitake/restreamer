/* Copyright (c) 2017 Gregor Riepl
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package restreamer

import (
	"errors"
	"fmt"
	"github.com/onitake/restreamer/api"
	"github.com/onitake/restreamer/util"
	"hash/fnv"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"time"
)

const (
	proxyBufferSize   = 1024
	proxyDefaultLimit = 10 * 1024 * 1024
	proxyDefaultMime  = "application/octet-stream"
	proxyFetchQueue   = 10
	//
	moduleProxy = "proxy"
	//
	eventProxyError           = "error"
	eventProxyStart           = "start"
	eventProxyShutdown        = "shutdown"
	eventProxyRequest         = "request"
	eventProxyOffline         = "offline"
	eventProxyFetch           = "fetch"
	eventProxyFetched         = "fetched"
	eventProxyRequesting      = "requesting"
	eventProxyRequestDone     = "requestdone"
	eventProxyReplyNotChanged = "replynotchanged"
	eventProxyReplyContent    = "replycontent"
	eventProxyStale           = "stale"
	eventProxyReturn          = "return"
	//
	errorProxyInvalidUrl    = "invalidurl"
	errorProxyNoLength      = "nolength"
	errorProxyLimitExceeded = "limitexceeded"
	errorProxyShortRead     = "shortread"
	errorProxyGet           = "get"
)

var (
	// headerList is a list of HTTP headers that are allowed to be sent through the proxy.
	headerList = []string{
		"Content-Type",
	}
	ErrNoLength      = errors.New("restreamer: Fetching of remote resource with unknown length not supported")
	ErrLimitExceeded = errors.New("restreamer: Resource too large for cache")
	ErrShortRead     = errors.New("restreamer: Short read, not all data was transferred in one go")
)

// fetchableResource contains a cachable resource and its metadata.
// This encapsulated type is used to ship data between the fetcher and the server.
type fetchableResource struct {
	// contents
	data []byte
	// upstream status code
	statusCode int
	// resource hash received from upstream, or computed
	etag string
	// upstream headers
	header http.Header
	// last update time (for aging)
	updated time.Time
}

// Proxy implements a caching HTTP proxy.
type Proxy struct {
	// the upstream URL (file/http/https)
	url *url.URL
	// HTTP client timeout
	timeout time.Duration
	// the cache time
	stale time.Duration
	// maximum size of remote resource
	limit int64
	// fetcher data request channel
	fetcher chan chan<- *fetchableResource
	// the cached resource
	// NOTE do not access this dirctly, use the fetcher instead
	resource *fetchableResource
	// a channel to signal shutdown to the fetcher
	shutdown chan struct{}
	// the global stats collector
	stats api.Statistics
	// a json logger
	logger *util.ModuleLogger
}

// NewProxy constructs a new HTTP proxy.
// The upstream resource is not fetched until the first request.
// If cache is non-zero, the resource will be evicted from memory after these
// number of seconds. If it is zero, the resource will be fetched from upstream
// every time it is requested.
// timeout sets the upstream HTTP connection timeout.
func NewProxy(uri string, timeout uint, cache uint) (*Proxy, error) {
	logger := &util.ModuleLogger{
		Logger: &util.ConsoleLogger{},
		Defaults: util.Dict{
			"module": moduleProxy,
			"uri":    uri,
		},
		AddTimestamp: true,
	}

	parsed, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	return &Proxy{
		url:     parsed,
		timeout: time.Duration(timeout) * time.Second,
		stale:   time.Duration(cache) * time.Second,
		// TODO make this configurable
		limit: proxyDefaultLimit,
		// TODO make queue length configurable
		fetcher:  make(chan chan<- *fetchableResource, proxyFetchQueue),
		shutdown: make(chan struct{}),
		resource: nil,
		stats:    &api.DummyStatistics{},
		logger:   logger,
	}, nil
}

// SetLogger assigns a logger.
func (proxy *Proxy) SetLogger(logger util.JsonLogger) {
	proxy.logger.Logger = logger
}

// SetStatistics assigns a stats collector.
func (proxy *Proxy) SetStatistics(stats api.Statistics) {
	proxy.stats = stats
}

// Get opens a remote or local resource specified by URL and returns a reader,
// upstream HTTP headers, an HTTP status code and the resource data length, or -1 if no length is available.
// Local resources contain guessed data.
// Supported protocols: file, http and https.
func Get(url *url.URL, timeout time.Duration) (reader io.Reader, header http.Header, status int, length int64, err error) {
	status = http.StatusNotFound
	reader = nil
	header = make(http.Header)
	err = nil
	length = 0

	if url.Scheme == "file" {
		reader, err = os.Open(url.Path)
		if err == nil {
			status = http.StatusOK

			// guess the size
			info, err2 := os.Stat(url.Path)
			if err2 == nil {
				length = info.Size()
			} else {
				// we can't stat, so the length is indefinite...
				length = -1
			}

			// guess the mime type
			mtype := mime.TypeByExtension(path.Ext(url.Path))
			if mtype != "" {
				header.Set("Content-Type", mtype)
			}
		} else {
			if err == os.ErrPermission {
				status = http.StatusForbidden
			} else {
				status = http.StatusNotFound
			}
		}
	} else {
		getter := &http.Client{
			Timeout: timeout,
		}
		response, err := getter.Get(url.String())
		if err == nil {
			status = response.StatusCode
			reader = response.Body
			length = response.ContentLength
			header = response.Header
		} else {
			// TODO: check if we have a timeout and return other status codes here
			status = http.StatusBadGateway
		}
	}

	return reader, header, status, length, err
}

// Start launches the fetcher thread.
// This should only be called once.
func (proxy *Proxy) Start() {
	proxy.logger.Log(util.Dict{
		"event":   eventProxyStart,
		"message": "Starting fetcher",
	})
	go proxy.fetch()
}

// Shutdown stops the fetcher thread.
func (proxy *Proxy) Shutdown() {
	proxy.logger.Log(util.Dict{
		"event":   eventProxyShutdown,
		"message": "Shutting down fetcher",
	})
	close(proxy.shutdown)
}

// fetch waits for fetch requests and handles them one-by-one.
// If the resource is already cached and not stale, it replies very quickly.
// Performance impact should be minimal in this case.
// Blocks while the resource is fetched.
func (proxy *Proxy) fetch() {
	running := true
	for running {
		select {
		case <-proxy.shutdown:
			running = false
		case request := <-proxy.fetcher:
			proxy.logger.Log(util.Dict{
				"event":    eventProxyRequest,
				"message":  "Handling request",
				"resource": proxy.resource,
			})
			// verify if we need to refetch
			now := time.Now()
			if proxy.resource == nil || now.Sub(proxy.resource.updated) > proxy.stale {
				// stale, cache first
				proxy.logger.Log(util.Dict{
					"event":   eventProxyStale,
					"message": "Resource is stale",
				})
				proxy.resource = proxy.cache()
			}
			// and return
			proxy.logger.Log(util.Dict{
				"event":   eventProxyReturn,
				"message": "Returning resource",
			})
			request <- proxy.resource
		}
	}
	proxy.logger.Log(util.Dict{
		"event":   eventProxyOffline,
		"message": "Fetcher is offline",
	})
}

// eTag calculates a hash value of data and returns it as a hex string.
// Suitable for HTTP Etags.
func Etag(data []byte) string {
	// 64-bit FNV-1a checksum of the data, formatted as a hex string
	hash := fnv.New64a()
	hash.Write(data)
	return fmt.Sprintf("%016x", hash.Sum64())
}

// cache fetches the remote resource into memory.
// Does not return errors. Instead, the cached resource contains a suitable return code and error content.
func (proxy *Proxy) cache() *fetchableResource {
	proxy.logger.Log(util.Dict{
		"event":   eventProxyFetch,
		"message": "Fetching resource from upstream",
	})

	// fetch from upstream
	getter, header, status, length, err := Get(proxy.url, proxy.timeout)
	if err != nil {
		proxy.logger.Log(util.Dict{
			"event":   eventProxyError,
			"error":   errorProxyGet,
			"message": err.Error(),
		})
	}

	// construct the return value
	res := &fetchableResource{
		header:     header,
		statusCode: status,
	}

	if err == nil {
		// verify the length
		if length < 0 {
			// TODO maybe allow caching of resources without length?
			err = ErrNoLength
			proxy.logger.Log(util.Dict{
				"event":   eventProxyError,
				"error":   errorProxyNoLength,
				"message": ErrNoLength,
			})
			res.statusCode = http.StatusBadGateway
			res.data = []byte(http.StatusText(res.statusCode))
			res.header = make(http.Header)
		} else if length > proxy.limit {
			err = ErrLimitExceeded
			proxy.logger.Log(util.Dict{
				"event":   eventProxyError,
				"error":   errorProxyLimitExceeded,
				"message": ErrLimitExceeded,
			})
			res.statusCode = http.StatusBadGateway
			res.data = []byte(http.StatusText(res.statusCode))
			res.header = make(http.Header)
		}
	}

	if err == nil {
		res.data = make([]byte, length)

		// fetch the data
		// TODO we should probably read in chunks
		var bytes int
		bytes, err = getter.Read(res.data)

		if err == nil && int64(bytes) != length {
			err = ErrShortRead
			proxy.logger.Log(util.Dict{
				"event":    eventProxyError,
				"error":    errorProxyShortRead,
				"message":  ErrShortRead,
				"length":   length,
				"received": bytes,
			})
			res.data = res.data[:bytes]
		}
	}

	res.updated = time.Now()
	// calculate the content hash
	res.etag = Etag(res.data)

	proxy.logger.Log(util.Dict{
		"event":   eventProxyFetched,
		"message": "Fetched resource from upstream",
		"etag":    res.etag,
		"length":  len(res.data),
		"status":  res.statusCode,
	})

	return res
}

// ServeHTTP handles an incoming connection.
// Satisfies the http.Handler interface, so it can be used in an HTTP server.
func (proxy *Proxy) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	// create a return channel for the fetcher
	fetchable := make(chan *fetchableResource)

	// request and wait for completion
	// since the channels are unbuffered, they will block on read/write
	// timeout must happen in the fetcher!
	proxy.logger.Log(util.Dict{
		"event":   eventProxyRequesting,
		"message": "Handling incoming request",
	})
	proxy.fetcher <- fetchable
	proxy.logger.Log(util.Dict{
		"event":   "waiting",
		"message": "Waiting for response",
	})
	res := <-fetchable
	close(fetchable)
	proxy.logger.Log(util.Dict{
		"event":   eventProxyRequestDone,
		"message": "Request complete",
	})

	// copy (appropriate) headers
	for _, key := range headerList {
		value := res.header.Get(key)
		if value != "" {
			writer.Header().Set(key, value)
		}
	}

	// headers for cached data
	writer.Header().Set("ETag", res.etag)
	// TODO maybe use the actual resource stale time here (Since())
	// TODO no-cache for errors!
	writer.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", int(proxy.stale.Seconds())))

	// verify if ETag has matched
	if res.etag != "" && request.Header.Get("If-None-Match") == res.etag {
		proxy.logger.Log(util.Dict{
			"event":   eventProxyReplyNotChanged,
			"message": "Returning 304",
		})
		// send only a 304
		writer.WriteHeader(http.StatusNotModified)
		// no content here
	} else {
		proxy.logger.Log(util.Dict{
			"event":   eventProxyReplyContent,
			"message": "Returning updated content",
			"updated": res.updated,
		})
		// otherwise, send updated data
		writer.Header().Set("Content-Length", strconv.Itoa(len(res.data)))
		writer.WriteHeader(res.statusCode)
		// and push the content
		writer.Write(res.data)
	}
}
