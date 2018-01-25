// +build !windows

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

package util

import (
	"os"
	"os/signal"
	"syscall"
)

const (
	// UserSignal is a unique identifier for the signal that is sent through the
	// notification channel when a user event occurs.
	UserSignal syscall.Signal = syscall.SIGUSR1
)

// RegisterUserSignalHandler registers a process signal handler that reacts to
// and notifies on external user events,  like SIGUSR1 on Unix.
// NOTE: Unsupported and ignored on Microsoft Windows.
func RegisterUserSignalHandler(notify chan os.Signal) {
	signal.Notify(notify, UserSignal)
}
