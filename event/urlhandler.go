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
	"net/url"
	"net/http"
)

type UrlHandler struct {
	Url *url.URL
}

func NewUrlHandler(urly string) (*UrlHandler, error) {
	u, err := url.Parse(urly)
	if err == nil {
		return &UrlHandler{u}, nil
	} else {
		return nil, err
	}
}

func (handler *UrlHandler) HandleEvent(EventType, ...interface{}) {
	req := &http.Request{
		Method: "GET",
		URL: handler.Url,
	}
	http.DefaultClient.Do(req)
}
