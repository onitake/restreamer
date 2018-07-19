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
	"net/http"
)

// HandleHttpAuthentication handles authentication headers and responses.
// If it returns false, authenticaten has failed, an appropriate response was sent and the caller should immediately return.
// A true return value indicates that authentication has succeeded and the caller should proceed with handling the request.
func HandleHttpAuthentication(auth Authenticator, request *http.Request, writer http.ResponseWriter) bool {
	// fail-fast: verify that this user can access this resource first
	if !auth.Authenticate(request.Header.Get("Authorization")) {
		realm := auth.GetAuthenticateRequest()
		if len(realm) > 0 {
			// if the authenticator supports responses to invalid authentication headers, send
			writer.Header().Add("WWW-Authenticate", realm)
			writer.WriteHeader(http.StatusUnauthorized)
		} else {
			// otherwise, just respond with a 403
			writer.WriteHeader(http.StatusForbidden)
		}
		return false
	}
	return true
}
