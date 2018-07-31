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

package event

import (
	"github.com/onitake/restreamer/util"
)

const (
	moduleEvent = "event"
	//
	queueEventError     = "error"
	queueEventLimitHit  = "hit"
	queueEventLimitMiss = "miss"
	queueEventStarting  = "starting"
	queueEventStopping  = "stopping"
	queueEventStarted   = "started"
	queueEventReceived  = "received"
	queueEventDraining  = "draining"
	queueEventStopped   = "stopped"
	queueEventConnect   = "connect"
	//
	queueErrorAlreadyRunning      = "already_running"
	queueErrorInvalidNotification = "invalid_notification"
	queueErrorUnderflow           = "underflow"
	queueErrorOverflow            = "overflow"
	queueErrorRegister            = "register"
	queueErrorNotRegistered       = "not_registered"
	//
	urlHandlerEventError  = "error"
	urlHandlerEventNotify = "notify"
	//
	urlHandlerErrorGet = "get"
)

var logger util.Logger = util.NewGlobalModuleLogger(moduleEvent, nil)
