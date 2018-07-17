/* Copyright (c) 2018 Gregor Riepl
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

package streaming

import (
	"encoding/base64"
	"strings"
	// 	"crypto/md5"
)

// Authenticator represents any type that can authenticate users.
type Authenticator interface {
	// Authenticate parses an Authorization header and tries to authenticate the request.
	// Returns true if the authentication succeeded, false otherwise.
	Authenticate(authorization string) bool
	// Adds a new user to the list.
	// Implementations may interpret users and passwords differently.
	AddUser(user, password string)
	// Removes a user from the list.
	// Implementations may interpret users differently.
	RemoveUser(user string)
}

// NewAuthenticator creates an authentication service from a credential datavase and
// an authentication specification. The implementation depends on the algorithm.
// If an invalid authentication type is specified, nil is returned.
// Empty whitelists allow no users at all!
// Note that some authenticators allow modifying the user list.
func NewAuthenticator(auth Authentication, credentials map[string]UserCredentials) Authenticator {
	switch auth.Type {
	case "basic":
		return newBasicAuthenticator(auth.Users, credentials)
	case "bearer":
		return newTokenAuthenticator(auth.Users, credentials)
	default:
		return nil
	}
}

type basicAuthenticator struct {
	// tokens maps valid authentication strings to yes/no
	tokens map[string]bool
	// users maps user names to valid authentication strings
	users map[string]string
}

// newBasicAuthenticator creates a new Authenticator that supports basic authentication.
// If the whitelist is empty, all requests are allowed.
func newBasicAuthenticator(whitelist []string, credentials map[string]UserCredentials) *basicAuthenticator {
	auth := &basicAuthenticator{
		tokens: make(map[string]bool),
		users:  make(map[string]string),
	}
	for _, user := range whitelist {
		cred, ok := credentials[user]
		if ok {
			auth.AddUser(user, cred.Password)
		}
	}
	return auth
}

func (auth *basicAuthenticator) Authenticate(authorization string) bool {
	if strings.HasPrefix(authorization, "Basic") {
		// cut off the hash at the end
		hash := strings.SplitN(authorization, " ", 2)
		if len(hash) >= 2 {
			// check if the hash is allowed
			return auth.tokens[hash[1]]
		}
	}
	// not basic auth
	return false
}

func (auth *basicAuthenticator) AddUser(user, password string) {
	// remove the old token if the user exists already
	if oldtoken, ok := auth.users[user]; ok {
		delete(auth.tokens, oldtoken)
	}
	// base64(username + ':' + password)
	// we only support UTF-8
	token := base64.StdEncoding.EncodeToString([]byte(user + ":" + password))
	auth.tokens[token] = true
	auth.users[user] = token
}

func (auth *basicAuthenticator) RemoveUser(user string) {
	token, ok := auth.users[user]
	if ok {
		delete(auth.users, user)
		delete(auth.tokens, token)
	}
}

type tokenAuthenticator struct {
	// tokens maps valid authentication tokens to yes/no
	tokens map[string]bool
	// users maps user names to valid authentication tokens
	users map[string]string
}

// newTokenAuthenticator creates a new Authenticator that supports bearer token authentication.
// The user name is only used as a unique identifier for the token list
func newTokenAuthenticator(whitelist []string, credentials map[string]UserCredentials) *tokenAuthenticator {
	auth := &tokenAuthenticator{
		tokens: make(map[string]bool),
		users:  make(map[string]string),
	}
	for _, user := range whitelist {
		cred, ok := credentials[user]
		if ok {
			auth.AddUser(user, cred.Password)
		}
	}
	return auth
}

func (auth *tokenAuthenticator) Authenticate(authorization string) bool {
	if strings.HasPrefix(authorization, "Bearer") {
		// cut off the hash at the end
		hash := strings.SplitN(authorization, " ", 2)
		if len(hash) >= 2 {
			// check if the hash is allowed
			return auth.tokens[hash[1]]
		}
	}
	// not basic auth
	return false
}

func (auth *tokenAuthenticator) AddUser(user, password string) {
	// remove the old token if the user exists already
	if oldtoken, ok := auth.users[user]; ok {
		delete(auth.tokens, oldtoken)
	}
	// base64(password)
	// we expect that token is already base64 formatted - do nothing here
	auth.tokens[password] = true
	auth.users[user] = password
}

func (auth *tokenAuthenticator) RemoveUser(user string) {
	token, ok := auth.users[user]
	if ok {
		delete(auth.users, user)
		delete(auth.tokens, token)
	}
}
