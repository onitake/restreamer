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

import (
	"github.com/onitake/restreamer/util"
)

const (
	moduleStreaming = "streaming"
	//
	eventAclError    = "error"
	eventAclAccepted = "accepted"
	eventAclDenied   = "denied"
	eventAclRemoved  = "removed"
	//
	errorAclNoConnection = "noconnection"
	//
	eventClientDebug            = "debug"
	eventClientError            = "error"
	eventClientRetry            = "retry"
	eventClientConnecting       = "connecting"
	eventClientConnectionLoss   = "loss"
	eventClientConnectTimeout   = "connect_timeout"
	eventClientOffline          = "offline"
	eventClientStarted          = "started"
	eventClientStopped          = "stopped"
	eventClientOpenPath         = "open_path"
	eventClientOpenHttp         = "open_http"
	eventClientOpenTcp          = "open_tcp"
	eventClientOpenDomain       = "open_domain"
	eventClientPull             = "pull"
	eventClientClosed           = "closed"
	eventClientTimerStop        = "timer_stop"
	eventClientTimerStopped     = "timer_stopped"
	eventClientNoPacket         = "nopacket"
	eventClientTimerKill        = "killed"
	eventClientReadTimeout      = "read_timeout"
	eventClientOpenUdp          = "open_udp"
	eventClientOpenUdpMulticast = "open_multicast"
	eventClientOpenFork         = "open_fork"
	//
	errorClientConnect       = "connect"
	errorClientParse         = "parse"
	errorClientInterface     = "interface"
	errorClientSetBufferSize = "buffersize"
	//
	eventConnectionDebug      = "debug"
	eventConnectionError      = "error"
	eventHeaderSent           = "headersent"
	eventConnectionClosed     = "closed"
	eventConnectionClosedWait = "closedwait"
	eventConnectionShutdown   = "shutdown"
	eventConnectionDone       = "done"
	//
	errorConnectionNotFlushable  = "noflush"
	errorConnectionNoCloseNotify = "noclosenotify"
	//
	eventProxyError           = "error"
	eventProxyStart           = "start"
	eventProxyShutdown        = "shutdown"
	eventProxyRequest         = "request"
	eventProxyOffline         = "offline"
	eventProxyFetch           = "fetch"
	eventProxyFetched         = "fetched"
	eventProxyRequesting      = "requesting"
	eventProxyRequestDone     = "requestdone"
	eventProxyReplyNotChanged = "replynotchanged"
	eventProxyReplyContent    = "replycontent"
	eventProxyStale           = "stale"
	eventProxyReturn          = "return"
	//
	errorProxyInvalidUrl      = "invalidurl"
	errorProxyNoLength        = "nolength"
	errorProxyLimitExceeded   = "limitexceeded"
	errorProxyShortRead       = "shortread"
	errorProxyGet             = "get"
	eventStreamerError        = "error"
	eventStreamerQueueStart   = "queuestart"
	eventStreamerStart        = "start"
	eventStreamerStop         = "stop"
	eventStreamerClientAdd    = "add"
	eventStreamerClientRemove = "remove"
	eventStreamerStreaming    = "streaming"
	eventStreamerClosed       = "closed"
	eventStreamerInhibit      = "inhibit"
	eventStreamerAllow        = "allow"
	//
	errorStreamerInvalidCommand = "invalidcmd"
	errorStreamerPoolFull       = "poolfull"
	errorStreamerOffline        = "offline"
)

var logger = util.NewGlobalModuleLogger(moduleStreaming, nil)
