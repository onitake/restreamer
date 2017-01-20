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
	"log"
	"fmt"
	"sync"
	"time"
	"net/http"
	"net/url"
	"hash/fnv"
)

// Proxy implements a caching HTTP proxy.
type Proxy struct {
	// the upstream URL (file/http/https)
	url *url.URL
	// fetch lock
	lock sync.RWMutex
	// the cache time
	cache time.Duration
	// the last fetch time
	last time.Time
	// the upstream status code
	status int
	// cache ETag to reduce traffic
	etag string
	// the cached data
	data []byte
}

// NewProxy constructs a new HTTP proxy.
// The upstream resource is not fetched until the first request.
// If cache is non-zero, the resource will be evicted from memory after these
// number of seconds. If it is zero, the resource will be fetched from upstream
// every time it is requested.
func NewProxy(uri string, cache uint, stats Statistics) (*Proxy, error) {
	parsed, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	
	return &Proxy{
		url: parsed,
		cache: time.Duration(cache) * time.Second,
		// not perfect, will cause problems if epoch is 0.
		// but it's very unlikely this will ever be a problem.
		// otherwise, add a "dirty" flag that tells when the resource needs to be fetched.
		last: time.Unix(0, 0),
	}, nil
}

func (proxy *Proxy) fetch() error {
	// acquire the write lock first
	proxy.lock.Lock()
	
	// double check to avoid a double fetch race
	now := time.Now()
	if now.Sub(proxy.last) > proxy.cache {
		// TODO actually fetch the resource here
		proxy.data = make([]byte, 0)
		
		// TODO set the status
		proxy.status = http.StatusOK
		// update the last fetch time
		proxy.last = now
		// TODO update the ETag only on valid content
		// calculate the ETag as the 64-bit FNV-1a checksum of the data
		hash := fnv.New64a()
		hash.Write(proxy.data)
		proxy.etag = fmt.Sprintf("%08x", hash.Sum64())
	}
	
	proxy.lock.Unlock()
	return nil
}

// ServeHTTP handles an incoming connection.
// Satisfies the http.Handler interface, so it can be used in an HTTP server.
func (proxy *Proxy) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	// lock first
	proxy.lock.RLock()
	
	// check if we need to fetch
	if time.Since(proxy.last) > proxy.cache {
		// not so nice, augmenting the read lock to a write lock would be nicer.
		// but there's no such luxury, so we need to unlock first and check
		// again after we've acquired the write lock.
		proxy.lock.RUnlock()
		
		// fetch and handle
		err := proxy.fetch()
		if err != nil {
			log.Print(err)
		}
		
		// and get back to serving
		proxy.lock.RLock()
	} else {
	}
	
	// verify if ETag has matched
	if proxy.etag != "" && request.Header.Get("If-None-Match") == proxy.etag {
		// always send caching headers if ETag match is requested and available
		writer.Header().Add("ETag", proxy.etag)
		writer.Header().Add("Cache-Control", fmt.Sprintf("max-age=%d", proxy.cache.Seconds()))
		writer.WriteHeader(http.StatusNotModified)
		
		// no content on a 304
	} else {
		// otherwise, only send headers when caching is enabled
		if proxy.cache > 0 {
			writer.Header().Add("ETag", proxy.etag)
			writer.Header().Add("Cache-Control", fmt.Sprintf("max-age=%d", proxy.cache.Seconds()))
		}
		writer.WriteHeader(proxy.status)

		// and push the content
		writer.Write(proxy.data)
	}
	
	proxy.lock.RUnlock()
}
