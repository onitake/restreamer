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

package protocol

import (
 	"encoding/base64"
	"testing"
	"math/rand"
	"github.com/onitake/restreamer/configuration"
)

const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-+.:,;$!#@%&/()=?'[]{}_<>"
// https://stackoverflow.com/a/31832326
func randStringBytes(n int) string {
    b := make([]byte, n)
    for i := range b {
        b[i] = alphabet[rand.Intn(len(alphabet))]
    }
    return string(b)
}

func TestPassAuthenticator01(t *testing.T) {
	auth := newPassAuthenticator()
	if !auth.Authenticate(randStringBytes(16)) {
		t.Errorf("Cannot pass random auth string")
	}
}

func TestDenyAuthenticator01(t *testing.T) {
	auth := newDenyAuthenticator()
	auth.AddUser("user", "password")
	str := base64.StdEncoding.EncodeToString([]byte("user:password"))
	if auth.Authenticate("Basic " + str) {
		t.Errorf("Deny authenticator incorrectly allowed basic auth")
	}
}

func TestBasicAuthenticator01(t *testing.T) {
	user := "user"
	password := randStringBytes(16)
	realm := "Test Realm"
	whitelist := []string{
		user,
	}
	cred := map[string]configuration.UserCredentials{
		user: configuration.UserCredentials{
			Password: password,
		},
	}
	auth := newBasicAuthenticator(whitelist, cred, realm)
	str := base64.StdEncoding.EncodeToString([]byte(user + ":" + password))
	if !auth.Authenticate("Basic " + str) {
		t.Errorf("Basic authenticator didn't allow valid user")
	}
}

func TestBasicAuthenticator02(t *testing.T) {
	user := "user"
	password := randStringBytes(16)
	realm := "Test Realm"
	whitelist := []string{}
	cred := map[string]configuration.UserCredentials{
		user: configuration.UserCredentials{
			Password: password,
		},
	}
	auth := newBasicAuthenticator(whitelist, cred, realm)
	str := base64.StdEncoding.EncodeToString([]byte(user + ":" + password))
	if auth.Authenticate("Basic " + str) {
		t.Errorf("Basic authenticator allowed non-whitelisted user")
	}
}
