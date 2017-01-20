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
	"time"
	"net/http"
	"net/url"
	"sync"
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
	
	// double check to avoid a double fetch
	now := time.Now()
	if now.Sub(proxy.last) > proxy.cache {
		// TODO actually fetch the resource here
		
		// update the last fetch time
		proxy.last = now
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
		// but there's no such luxury...
		proxy.lock.RUnlock()
		proxy.fetch()
		// and get back to reading (i.e. serving again)
		proxy.lock.RLock()
	}
	
	// and push the content
	// TODO handle upstream errors
	writer.WriteHeader(http.StatusOK)
	writer.Write(proxy.data)
	
	proxy.lock.RUnlock()
}
