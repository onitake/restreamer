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

import ()

type UserTable struct {
	credentials map[string]UserCredentials
	whitelist   Authentication
}

// MakeUserTable creates a user verification table from user credentials and a whitelist.
func MakeUserTable(credentials map[string]UserCredentials, whitelist Authentication) *UserTable {
	table := &UserTable{
		credentials: credentials,
		whitelist:   whitelist,
	}
	return table
}

// Authenticate authenticates a user by parsing and matching an Authorization header
// against the user table.
// Returns true if the authentication succeeded, false otherwise.
// If the whitelist is empty, authentication will always succeed.
func (table *UserTable) Authenticate(authorization string) bool {
	if len(table.whitelist.Users) == 0 {
		return true
	}
	// TODO
	return false
}
