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

package auth

import (
	"encoding/base64"
	"strings"
	// 	"crypto/md5"
	"github.com/onitake/restreamer/configuration"
)

// Authenticator represents any type that can authenticate users.
type Authenticator interface {
	// Authenticate parses an Authorization header and tries to authenticate the request.
	// Returns true if the authentication succeeded, false otherwise.
	Authenticate(authorization string) bool
	// AddUser adds a new user to the list.
	// Implementations may interpret users and passwords differently.
	AddUser(user, password string)
	// RemoveUser removes a user from the list.
	// Implementations may interpret users differently.
	RemoveUser(user string)
	// GetLogin returns an authentication string that can be sent to a remote system.
	GetLogin(user string) string
	// GetAuthenticateRequest returns a realm or other response that can be sent with a WWW-Authenticate header.
	GetAuthenticateRequest() string
}

// NewAuthenticator creates an authentication service from a credential database and
// an authentication specification. The implementation depends on the algorithm.
//
// If an invalid authentication type is specified, an authenticator that will always
// deny requests is returned.
// If an empty authentication type is specified, an authenticator that will accept
// all requests is returned.
//
// Note: Empty whitelists allow no users at all!
func NewAuthenticator(auth configuration.Authentication, credentials map[string]configuration.UserCredentials) Authenticator {
	switch auth.Type {
	case "":
		return newPassAuthenticator()
	case "basic":
		return newBasicAuthenticator(auth.Users, credentials, auth.Realm)
	case "bearer":
		return newTokenAuthenticator(auth.Users, credentials)
	default:
		return newDenyAuthenticator()
	}
}

type passAuthenticator struct{}

func newPassAuthenticator() *passAuthenticator {
	return &passAuthenticator{}
}

func (auth *passAuthenticator) Authenticate(authorization string) bool {
	return true
}

func (auth *passAuthenticator) AddUser(user, password string) {}

func (auth *passAuthenticator) RemoveUser(user string) {}

func (auth *passAuthenticator) GetLogin(user string) string {
	return ""
}
func (auth *passAuthenticator) GetAuthenticateRequest() string {
	return ""
}

type denyAuthenticator struct{}

func newDenyAuthenticator() *denyAuthenticator {
	return &denyAuthenticator{}
}

func (auth *denyAuthenticator) Authenticate(authorization string) bool {
	return false
}

func (auth *denyAuthenticator) AddUser(user, password string) {}

func (auth *denyAuthenticator) RemoveUser(user string) {}

func (auth *denyAuthenticator) GetLogin(user string) string {
	return ""
}
func (auth *denyAuthenticator) GetAuthenticateRequest() string {
	return ""
}

type basicAuthenticator struct {
	// tokens maps valid authentication strings to yes/no
	tokens map[string]bool
	// users maps user names to valid authentication strings
	users map[string]string
	// the authentication realm (unique string sent back with an unauthorized response
	realm string
}

// newBasicAuthenticator creates a new Authenticator that supports basic authentication.
// If the whitelist is empty, no requests are allowed.
func newBasicAuthenticator(whitelist []string, credentials map[string]configuration.UserCredentials, realm string) *basicAuthenticator {
	auth := &basicAuthenticator{
		tokens: make(map[string]bool),
		users:  make(map[string]string),
		realm:  realm,
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

func (auth *basicAuthenticator) GetLogin(user string) string {
	if token, ok := auth.users[user]; ok {
		return "Basic " + token
	}
	return ""
}

func (auth *basicAuthenticator) GetAuthenticateRequest() string {
	return "Basic realm=\"" + auth.realm + "\" charset=\"UTF-8\""
}

type tokenAuthenticator struct {
	// tokens maps valid authentication tokens to yes/no
	tokens map[string]bool
	// users maps user names to valid authentication tokens
	users map[string]string
}

// newTokenAuthenticator creates a new Authenticator that supports bearer token authentication.
// The user name is only used as a unique identifier for the token list
func newTokenAuthenticator(whitelist []string, credentials map[string]configuration.UserCredentials) *tokenAuthenticator {
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

func (auth *tokenAuthenticator) GetLogin(user string) string {
	if token, ok := auth.users[user]; ok {
		return "Bearer " + token
	}
	return ""
}

func (auth *tokenAuthenticator) GetAuthenticateRequest() string {
	// token-based auth doesn't support challenge-response or similar, as token are generated externally
	// just send back a 403.
	return ""
}

// UserAuthenticator is an authenticator that is bound to a single user.
// It does not implement the Authenticator interface because it doesn't support the user argument.
type UserAuthenticator struct {
	Auth Authenticator
	User string
}

// NewUserAuthenticator creates a new user authenticator from an Authentication configuration and
// and an authenticator.
// If the Authentication does not contain any users, nil is returned. If it contains more than
// one user, the first one is used.
func NewUserAuthenticator(cred configuration.Authentication, auth Authenticator) *UserAuthenticator {
	if len(cred.Users) < 1 {
		return nil
	}
	return &UserAuthenticator{
		Auth: auth,
		User: cred.Users[0],
	}
}

func (auth *UserAuthenticator) GetLogin() string {
	return auth.Auth.GetLogin(auth.User)
}
