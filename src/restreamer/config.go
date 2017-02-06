/* Copyright (c) 2016-2017 Gregor Riepl
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
	"encoding/json"
)

// Configuration is a representation of the configurable settings.
// These are normally read from a JSON file and deserialized by
// the builtin marshaler.
type Configuration struct {
	// the interface to listen on
	Listen string `json:"listen"`
	// the connection timeout
	// (both input and output)
	Timeout uint `json:"timeout"`
	// the reconnect delay
	Reconnect uint `json:"reconnect"`
	// the maximum number of packets
	// on the input buffer
	InputBuffer uint `json:"inputbuffer"`
	// the size of the output buffer
	// per connection
	// note that each connection will
	// eat at least OutputBuffer * 192 bytes
	// when the queue is full, so
	// you should adjust the value according
	// to the amount of RAM available
	OutputBuffer uint `json:"outputbuffer"`
	// the maximum total number of concurrent connections
	MaxConnections uint `json:"maxconnections"`
	// set to true to disable statistics
	NoStats bool `json:"nostats"`
	// the list of streams
	Resources []struct {
		// the resource type
		Type string `json:"type"`
		// the API type
		Api string `json:"api"`
		// the local URL to serve this stream under
		Serve string `json:"serve"`
		// a single upstream URL or API argument
		// will be added to Remotes during parsing
		Remote string `json:"remote"`
		// the upstream URLs
		Remotes []string `json:"remotes"`
		// the cache time in seconds
		Cache uint `json:"cache"`
	} `json:"resources"`
}

// DefaultConfiguration creates and returns a configuration object
// with default values.
func DefaultConfiguration() Configuration {
	return Configuration{
		Listen: "localhost:http",
		Timeout: 0,
		Reconnect: 10,
		InputBuffer: 1000,
		OutputBuffer: 400,
		MaxConnections: 1,
		NoStats: false,
	}
}

// LoadConfiguration loads a configuration in JSON format from "filename".
func LoadConfiguration(filename string) (Configuration, error) {
	config := DefaultConfiguration()
	
	fd, err := os.Open(filename)
	if err == nil {
		decoder := json.NewDecoder(fd)
		err = decoder.Decode(&config)
		fd.Close()
	}
	
	for i := range config.Resources {
		// add remote to remotes list, if given
		if len(config.Resources[i].Remote) > 0 {
			length := len(config.Resources[i].Remotes)
			remotes := make([]string, length + 1)
			remotes[0] = config.Resources[i].Remote
			copy(remotes[1:], config.Resources[i].Remotes)
			config.Resources[i].Remotes = remotes
		}
	}
	
	return config, err
}
