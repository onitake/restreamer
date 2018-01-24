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
	"os"
	"io"
	"log"
	"fmt"
	"sync"
	"time"
	"mime"
	"path"
	"errors"
	"strconv"
	"net/http"
	"net/url"
	"hash/fnv"
)

const (
	proxyBufferSize = 1024
	proxyDefaultLimit = 10*1024*1024
	proxyDefaultMime = "application/octet-stream"
)

var (
	// headerList is a list of HTTP headers that are allowed to be sent through the proxy.
	headerList = []string{
		"Content-Type",
	}
	ErrNoLength = errors.New("restreamer: Fetching of remote resource with unknown length not supported")
	ErrLimitExceeded = errors.New("restreamer: Resource too large for cache")
)

// Proxy implements a caching HTTP proxy.
type Proxy struct {
	// the upstream URL (file/http/https)
	url *url.URL
	// HTTP client timeout
	timeout time.Duration
	// maximum size of remote resource
	limit int64
	// fetch lock
	lock sync.RWMutex
	// the cache time
	stale time.Duration
	// the last fetch time
	last time.Time
	// the upstream status code
	status int
	// cache ETag to reduce traffic
	etag string
	// upstream headers
	header http.Header
	// the cached data
	data []byte
	// the global stats collector
	stats Statistics
	// a json logger
	logger JsonLogger
}

// NewProxy constructs a new HTTP proxy.
// The upstream resource is not fetched until the first request.
// If cache is non-zero, the resource will be evicted from memory after these
// number of seconds. If it is zero, the resource will be fetched from upstream
// every time it is requested.
// timeout sets the upstream HTTP connection timeout.
func NewProxy(uri string, timeout uint, cache uint) (*Proxy, error) {
	parsed, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	
	return &Proxy{
		url: parsed,
		timeout: time.Duration(timeout) * time.Second,
		// TODO make this configurable
		limit: proxyDefaultLimit,
		stale: time.Duration(cache) * time.Second,
		// not perfect, will cause problems if epoch is 0.
		// but it's very unlikely this will ever be a problem.
		// otherwise, add a "dirty" flag that tells when the resource needs to be fetched.
		last: time.Unix(0, 0),
		header: make(http.Header),
		stats: &DummyStatistics{},
		logger: &DummyLogger{},
	}, nil
}

// Assigns a logger
func (proxy *Proxy) SetLogger(logger JsonLogger) {
	proxy.logger = logger
}

// Assigns a stats collector
func (proxy *Proxy) SetStatistics(stats Statistics) {
	proxy.stats = stats
}

// Get opens the remote or local resource specified by the URL and returns a reader, 
// upstream HTTP headers, an HTTP status code and the resource data length, or -1 if no length is available.
// Local resources contain guessed data
// Supported schemas: file, http and https.
func Get(url *url.URL, timeout time.Duration) (io.Reader, http.Header, int, int64, error) {
	var status int = http.StatusNotFound
	var reader io.Reader = nil
	var header http.Header = make(http.Header)
	var err error = nil
	var length int64 = 0
	
	if url.Scheme == "file" {
		log.Printf("Fetching %s\n", url.Path)
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
		log.Printf("Fetching %s\n", url)
		getter := &http.Client {
			Timeout: timeout,
		}
		response, err := getter.Get(url.String())
		if (err == nil) {
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

// cache fetches the remote resource into memory.
func (proxy *Proxy) cache() error {
	// acquire the write lock first
	proxy.lock.Lock()
	
	// double check to avoid a double fetch race
	now := time.Now()
	if now.Sub(proxy.last) > proxy.stale {
		// get a getter
		getter, header, status, length, err := Get(proxy.url, proxy.timeout)
		
		// no length, no cache
		if length < 0 {
			// TODO maybe allow caching of resources without length?
			err = ErrNoLength
			length = 0
		}
		// too large, no cache
		if length > proxy.limit {
			err = ErrLimitExceeded
			length = 0
		}
		
		// update
		proxy.header = header
		proxy.status = status
		proxy.last = now
		proxy.data = make([]byte, length)
		
		if err == nil {
			// fetch the data
			var bytes int
			bytes, err = getter.Read(proxy.data)
			
			// TODO we should also support reading in multiple chunks
			if int64(bytes) != length {
				log.Printf("Short read, not all data was transferred in one go. expected=%d got=%d", length, bytes)
				proxy.data = proxy.data[:bytes]
			}
			
			if err == nil {
				// calculate the ETag as the 64-bit FNV-1a checksum of the data, formatted as a hex string
				hash := fnv.New64a()
				hash.Write(proxy.data)
				proxy.etag = fmt.Sprintf("%016x", hash.Sum64())
			}
		}
		
		if err != nil {
			return err
		}
	}
	
	proxy.lock.Unlock()
	return nil
}

// ServeHTTP handles an incoming connection.
// Satisfies the http.Handler interface, so it can be used in an HTTP server.
func (proxy *Proxy) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	// lock first
	proxy.lock.RLock()
	
	// handle cached and non-cached resources separately
	if proxy.stale > 0 {
		// cached
		var err error
		
		// check if we need to fetch
		if time.Since(proxy.last) > proxy.stale {
			// not so nice: augmenting the read lock to a write lock would be nicer.
			// but there's no such luxury, so we need to unlock first and check
			// again after we've acquired the write lock.
			proxy.lock.RUnlock()
			// cache the resource
			log.Printf("Resource %s is stale, re-fetching\n", proxy.url)
			err = proxy.cache()
			// and get back to serving
			proxy.lock.RLock()
		}
	
		if err != nil {
			log.Printf("Error fetching resource: %s", err)
		}
		
		// copy headers here
		for _, key := range headerList {
			value := proxy.header.Get(key)
			if value != "" {
				writer.Header().Set(key, value)
			}
		}
		
		// headers for cached data
		writer.Header().Set("ETag", proxy.etag)
		// TODO maybe use the actual resource stale time here (Since())
		writer.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", int(proxy.stale.Seconds())))
		
		// verify if ETag has matched
		if proxy.etag != "" && request.Header.Get("If-None-Match") == proxy.etag {
			// send only a 304
			writer.WriteHeader(http.StatusNotModified)
			// no content here
		} else {
			// otherwise, send updated data
			writer.Header().Set("Content-Length", strconv.Itoa(len(proxy.data)))
			writer.WriteHeader(proxy.status)
			// and push the content
			writer.Write(proxy.data)
		}
	} else {
		// non-cached
		
		// get a getter
		getter, header, status, length, err := Get(proxy.url, proxy.timeout)
		
		if err != nil {
			log.Printf("Error connecting to upstream: %s", err)
		}
		
		// write the header
		for _, key := range headerList {
			value := header.Get(key)
			if value != "" {
				writer.Header().Set(key, value)
			}
		}
		writer.Header().Set("Cache-Control", "no-cache")
		if length != -1 {
			writer.Header().Set("Content-Length", strconv.FormatInt(length, 10))
		}
		writer.WriteHeader(status)
		
		// and transfer the data
		buffer := make([]byte, proxyBufferSize)
		eof := false
		for !eof {
			bytes, err := getter.Read(buffer)
			log.Printf("Got %d bytes, error %s", bytes, err)
			if bytes > 0 {
				bytes, err = writer.Write(buffer[:bytes])
				log.Printf("Wrote %d bytes, error %s", bytes, err)
			} else {
				log.Printf("No data received, should we exit here?")
			}
			if err != nil {
				if err != io.EOF {
					log.Printf("Error sending data to client: %s", err)
				}
				eof = true
			}
		}
	}
	
	proxy.lock.RUnlock()
}
