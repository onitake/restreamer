/* Copyright (c) 2017 Gregor Riepl
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

package restreamer

// StateManager maintains a list of disconnectable objects, sending them
// a notification whenever state changes.
//
// After connection closure has been notified, the list is cleared and
// further notifications have no effect.
type StateManager struct {
	notifiables map[chan<- bool]bool
}

// NewStateManager creates a new state manager.
//
// Register notification channels with Register(), and submit state changes
// with Notify(). Channels can be removed later with Unregister().
// After notify has been called, the list of registered channels is cleared.
func NewStateManager() *StateManager {
	return &StateManager{
		notifiables: make(map[chan<- bool]bool),
	}
}

// Registers a new notification channel.
//
// It is not possible to register a channel twice.
// Any additional registrations will be ignored.
func (manager *StateManager) Register(channel chan<- bool) {
	manager.notifiables[channel] = true
}

// Removes a registered channel.
//
// If this channel was not registered previously, no action is taken.
func (manager *StateManager) Unregister(channel chan<- bool) {
	delete(manager.notifiables, channel)
}

// Sends a state change to all registered notification channels and clears the list.
func (manager *StateManager) Notify() {
	for notifiable, _ := range manager.notifiables {
		notifiable<- true
	}
	manager.notifiables = make(map[chan<- bool]bool)
}
