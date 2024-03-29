/* Copyright (c) 2022 Gregor Riepl
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
	"github.com/onitake/restreamer/util"
)

const (
	moduleProtocol = "protocol"
	//
	eventForkError        = "error"
	eventForkStarted      = "forked"
	eventForkChildMessage = "childmessage"
	//
	errorForkExit       = "exit_error"
	errorForkStderrRead = "stderr_read"
)

var logger = util.NewGlobalModuleLogger(moduleProtocol, nil)
