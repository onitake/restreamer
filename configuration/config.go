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

package configuration

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
)

// Authentication configures authentication for a resource.
// The exact semantics depend on the resource.
type Authentication struct {
	// Type specifies the authentication type.
	// Only the empty string, 'basic' and 'bearer' are currently supported.
	// The interpretation of the type is as follows:
	// '': Disable authentication and allow all requests to succeed.
	// 'basic': compare the string after the 'Authorization: Basic' header with
	// base64(md5sum(username + ':' + passwords[username])) and allow the request if they match.
	// 'bearer': compare the string after 'Authentication: Bearer' with
	// base64(passwords[username]) and allow the request if they match.
	Type string `json:"type"`
	// Realm specifies the authentication realm that is sent
	// back to the client if the authentication header was missing.
	Realm string `json:"realm"`
	// User specifies a valid user who can access the resource.
	// This is merged with Users.
	User string `json:"user"`
	// Users specifies the list of valid user names.
	// User is merged into this list.
	Users []string `json:"users"`
}

// Resource is a single HTTP endpoint.
type Resource struct {
	// Type is the resource type.
	Type string `json:"type"`
	// Api is the API type.
	Api string `json:"api"`
	// Serve is the local URL to serve this stream under.
	Serve string `json:"serve"`
	// Remote is a single upstream URL or API argument;
	// it will be added to Remotes during parsing.
	Remote string `json:"remote"`
	// Remotes is the upstream URLs.
	Remotes []string `json:"remotes"`
	// Cache the cache time in seconds.
	Cache uint `json:"cache"`
	// Authentication specifies credentials required to access this resource.
	// If the authentication type is unset, no authentication is required.
	Authentication Authentication `json:"authentication"`
}

// UserCredentials is a set of credentials for a single user
type UserCredentials struct {
	// Password is the key or password of this user.
	Password string `json:"password"`
}

// Notification is a single notification definition.
type Notification struct {
	// Event is the event to watch for.
	Event string `json:"event"`
	// Type is the kind of callback to send.
	Type string `json:"type"`
	// Url is the remote to access (if Type is http).
	Url string `json:"url"`
	// Authentication specifies credentials to be sent with the notification.
	// If the authentication type is unset, no authentication is sent.
	// Only the first user from the list (or the single 'User') is used, all others are ignored.
	Authentication Authentication `json:"authentication"`
}

// Configuration is a representation of the configurable settings.
// These are normally read from a JSON file and deserialized by
// the builtin marshaler.
type Configuration struct {
	// Listen is the interface to listen on.
	Listen string `json:"listen"`
	// Timeout is the connection timeout
	// (both input and output).
	Timeout uint `json:"timeout"`
	// Reconnect is the reconnect delay.
	Reconnect uint `json:"reconnect"`
	// ReadTimeout is the upstream read timeout.
	ReadTimeout uint `json:"readtimeout"`
	// InputBuffer is the maximum number of packets.
	// on the input buffer
	InputBuffer uint `json:"inputbuffer"`
	// OutputBuffer is the size of the output buffer per connection.
	// Note that each connection will eat at least OutputBuffer * 192 bytes
	// when the queue is full, so you should adjust the value according
	// to the amount of RAM available.
	OutputBuffer uint `json:"outputbuffer"`
	// MaxConnections is the maximum total number of concurrent connections.
	// If it is 0, no hard limit will be imposed.
	MaxConnections uint `json:"maxconnections"`
	// FullConnections is the soft limit on the total number of concurrent connections.
	// If it is 0, no soft limit will be imposed/reported.
	FullConnections uint `json:"fullconnections"`
	// NoStats disables statistics collection, if set.
	NoStats bool `json:"nostats"`
	// Log is the access log file name.
	Log string `json:"log"`
	// Profile determines if profiling should be enabled.
	// Set to true to turn on the pprof web server.
	Profile bool `json:"profile"`
	// UserList is the built-in list of user accounts, to be used with authentication stanzas.
	// It maps user names to authentication credentials.
	UserList map[string]UserCredentials `json:"userlist"`
	// Resources is the list of streams.
	Resources []Resource `json:"resources"`
	// Notifications defines event callbacks.
	Notifications []Notification `json:"notifications"`
}

// DefaultConfiguration creates and returns a configuration object
// with default values.
func DefaultConfiguration() *Configuration {
	return &Configuration{
		Listen:       "localhost:http",
		Timeout:      0,
		Reconnect:    10,
		InputBuffer:  1000,
		OutputBuffer: 400,
		NoStats:      false,
	}
}

// LoadConfigurationFile loads a configuration in JSON format from "filename".
func LoadConfigurationFile(filename string) (*Configuration, error) {
	fd, err := os.Open(filename)
	if err == nil {
		defer fd.Close()
		return LoadConfiguration(fd)
	} else {
		return nil, err
	}
}

// LoadConfiguration reads JSON data from the Reader argument and returns a parsed configuration from it.
func LoadConfiguration(reader io.Reader) (*Configuration, error) {
	config := DefaultConfiguration()

	decoder := json.NewDecoder(reader)
	err := decoder.Decode(&config)
	if err != nil {
		return nil, err
	}

	for i := range config.Resources {
		resource := &config.Resources[i]
		// add remote to remotes list, if given - but only if this is a stream
		if resource.Type == "stream" && len(resource.Remote) > 0 {
			length := len(resource.Remotes)
			remotes := make([]string, length+1)
			remotes[0] = resource.Remote
			copy(remotes[1:], resource.Remotes)
			resource.Remotes = remotes
			// reset
			resource.Remote = ""
		}
		// add user to users list, if given
		if len(resource.Authentication.Type) > 0 && len(resource.Authentication.User) > 0 {
			length := len(resource.Authentication.Users)
			users := make([]string, length+1)
			users[0] = resource.Authentication.User
			copy(users[1:], resource.Authentication.Users)
			resource.Authentication.Users = users
			// reset
			resource.Authentication.User = ""
		}
	}
	for i := range config.Notifications {
		notification := &config.Notifications[i]
		// add user to users list, if given
		if len(notification.Authentication.Type) > 0 && len(notification.Authentication.User) > 0 {
			length := len(notification.Authentication.Users)
			users := make([]string, length+1)
			users[0] = notification.Authentication.User
			copy(users[1:], notification.Authentication.Users)
			notification.Authentication.Users = users
			// reset
			notification.Authentication.User = ""
		}
	}

	return config, err
}

// LoadConfigurationBytes parses the byte array argument as JSON and initialises a configuration from it.
func LoadConfigurationBytes(json []byte) (*Configuration, error) {
	return LoadConfiguration(bytes.NewReader(json))
}
