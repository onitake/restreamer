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

package main

import (
	"github.com/onitake/restreamer/util"
)

const (
	moduleMain = "main"
	//
	eventMainError        = "error"
	eventMainConfig       = "config"
	eventMainConfigStream = "stream"
	eventMainConfigStatic = "static"
	eventMainConfigApi    = "api"
	eventMainHandled      = "handled"
	eventMainStartMonitor = "start_monitor"
	eventMainStartServer  = "start_server"
	//
	errorMainStreamNotFound          = "stream_notfound"
	errorMainInvalidApi              = "invalid_api"
	errorMainInvalidResource         = "invalid_resource"
	errorMainInvalidNotification     = "invalid_notification"
	errorMainMissingNotificationUser = "missing_notification_user"
	errorMainMissingStreamUser       = "missing_stream_user"
	errorMainInvalidAuthentication   = "invalid_authentication"
)

var logger util.JsonLogger = util.NewGlobalModuleLogger(moduleMain, nil)
