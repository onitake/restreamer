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
	"fmt"
	"github.com/onitake/restreamer/configuration"
	"github.com/onitake/restreamer/util"
	"net/http"
	"net/url"
)

const (
	moduleUrlHandler = "urlhandler"
	//
	urlHandlerEventError  = "error"
	urlHandlerEventNotify = "notify"
	//
	urlHandlerErrorGet = "get"
)

// UrlHandler is an event handler that can send GET requests to a preconfigured HTTP URL.
type UrlHandler struct {
	// Url is the parsed URL
	Url *url.URL
	// logger is a json logger
	logger *util.ModuleLogger
	// userauth will be used to generate credentials for client requests
	userauth *configuration.UserAuthenticator
}

func NewUrlHandler(urly string, userauth *configuration.UserAuthenticator) (*UrlHandler, error) {
	logger := &util.ModuleLogger{
		Logger: &util.ConsoleLogger{},
		Defaults: util.Dict{
			"module": moduleUrlHandler,
		},
		AddTimestamp: true,
	}
	u, err := url.Parse(urly)
	if err == nil {
		return &UrlHandler{
			Url:      u,
			logger:   logger,
			userauth: userauth,
		}, nil
	} else {
		return nil, err
	}
}

// SetLogger assigns a logger
func (handler *UrlHandler) SetLogger(logger util.JsonLogger) {
	handler.logger.Logger = logger
}

func (handler *UrlHandler) HandleEvent(typ EventType, args ...interface{}) {
	handler.logger.Log(util.Dict{
		"event":   urlHandlerEventNotify,
		"message": fmt.Sprintf("Event received, notifying %s", handler.Url),
		"url":     handler.Url.String(),
		"type":    typ,
	})
	req := &http.Request{
		Method: "GET",
		URL:    handler.Url,
	}
	_, err := http.DefaultClient.Do(req)
	if err != nil {
		handler.logger.Log(util.Dict{
			"event":   urlHandlerEventError,
			"error":   urlHandlerErrorGet,
			"message": fmt.Sprintf("Error sending GET request: %v", err),
			"url":     handler.Url.String(),
			"type":    typ,
		})
	}
}
