[![language](https://img.shields.io/badge/language-go-%2300ADD8.svg?style=flat-square)](https://golang.org/) [![release](https://img.shields.io/github/release/onitake/restreamer.svg?style=flat-square)](https://github.com/onitake/restreamer/releases) [![](https://img.shields.io/travis/onitake/restreamer.svg?style=flat-square)](https://travis-ci.org/onitake/restreamer)

![restreamer logo](doc/logo.png)

# restreamer

HTTP transport stream proxy

Copyright © 2016-2018 Gregor Riepl;
All rights reserved.

Please see the LICENSE file for details on permitted use of this software.


## Introduction

restreamer is a proof-of-concept implementation of a streaming proxy
that can fetch, buffer and distribute [MPEG transport streams](https://en.wikipedia.org/wiki/MPEG-TS).

It serves as a non-transcoding streaming proxy for legacy HbbTV applications.

Data sources can be local files, remote HTTP servers or raw TCP streams.
Unix domain sockets are also supported.

The proxy is stateless: Streams are transported in realtime
and cached resources are only kept in memory.
This makes restreamer perfectly suited for a multi-tiered architecture,
that can be easily deployed on containers and expanded or shrunk as
load demands.


## Architecture

restreamer is written in [Go](https://golang.org/), a very versatile
programming language. The built-in network layer makes it a first-class
choice for a streaming server.

These are the key components:
* util - a small utility library
* streaming/client - HTTP getter for fetching upstream data
* streaming/connection - HTTP server that feeds data to clients
* streaming/streamer - connection broker and data queue
* api/api - web API for service monitoring
* streaming/proxy - static web server and proxy
* protocol - network protocol library
* configuration - abstraction of the configuration file
* metrics - a small wrapper around the Promethus client library
* metrics/stats - the old, deprecated metrics collector; use Prometheus if possible
* cmd/restreamer - core program that glues the components together


## Compilation

restreamer is go-gettable.
Just invoke:

```
CGO_ENABLED=0 go get github.com/onitake/restreamer/...
```

Disabling CGO is recommended, as this will produce a standalone binary that
does not depend on libc. This is useful for running restreamer in a bare container.

A makefile is also provided, as a quick build reference.
Simply invoke `make` to build `restreamer`.

Passing GOOS and/or GOARCH will yield cross-compiled binaries for the
respective platform. For example:

```
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build github.com/onitake/restreamer/cmd/restreamer
```
or
```
make GOOS=windows GOARCH=amd64
```

You can also use `make test` to run the test suite, or `make fmt` to run `go fmt` on all sources.


## Configuration

All configuration is done through a configuration file named `restreamer.json`.

See `examples/documented/restreamer.json` for a documented example.

The input and output buffer sizes should be adapted to the expected stream
bitrates and must account for unstable or slow client-side internet connections.

Output buffers should cover at least 1-5 seconds of video and the input buffer
should be 4 times as much.
The amount of packets to buffer, based on the video+audio bitrates, can be
calculated with this formula (overhead is not taken into account):

```
mpegts_packet_size = 188
buffer_size_packets = (avg_video_bitrate + avg_audio_bitrate) * buffer_size_seconds / (mpegts_packet_size * 8)
```

It is also important to keep the bandwidth of the network interfaces
in mind, so the connection limit should be set accordingly.

Buffer memory usage is equally important, and can be roughly calculated as follows:

```
mpegts_packet_size = 188
max_buffer_memory = mpegts_packet_size * (number_of_streams * input_buffer_size + max_connections * output_buffer_size)
```

It is possible to specify multiple upstream URLs per stream.
These will be tested in a round-robin fashion in random order,
with the first successful one being used.
The list is only shuffled once, at program startup.

If a connection is terminated, all URLs will be tried again after a delay.
If delay is 0, the stream will stay offline.

To protect against overload, both a soft and a hard limit on the number of
downstream connections can be set. When the soft limit is reached, the health
API will start reporting that the server is "full". Once the hard limit is
reached, new connections will be responded with a 404. A 503 would be more
appropriate, but this is not handled well by many legacy streaming clients.


## Logging

restreamer has a JSON logging module built in.
Output can be sent to stdout or written to a log file.

It is highly recommended to log to stdout and collect logs using journald
or a similar logging engine.


## Metrics

Metrics are exported through the Prometheus client library. Enable a Prometheus
API endpoint and expose it on /metrics to expose them.

Supported metrics are:

* streaming_packets_sent
  Total number of MPEG-TS packets sent from the output queue.
* streaming_bytes_sent
  Total number of bytes sent from the output queue.
* streaming_packets_dropped
  Total number of MPEG-TS packets dropped from the output queue.
* streaming_bytes_dropped"
  Total number of bytes dropped from the output queue.
* streaming_connections
  Number of active client connections.
* streaming_duration
  Total time spent streaming, summed over all client connections. In nanoseconds.
* streaming_source_connected
  Connection status, 0=disconnected 1=connected.
* streaming_packets_received
  Total number of MPEG-TS packets received.
* streaming_bytes_received
  Total number of bytes received.

Additionally, the standard process metrics supported by the Prometheus client
library are exported. Go runtime statistics are disabled, as they can have a
considerable effect on realtime operation. To enable them, you need to turn on
profiling.


## Optimisation

### Test Stream

Using the example config and ffmpeg, a test stream can be set up.

Create a pipe:
```
mkfifo /tmp/pipe.ts
```
Start feeding data into the pipe (note the -re parameter for real-time streaming):
```
ffmpeg -re -stream_loop -1 -i test.ts -fflags +genpts -c:a copy -c:v copy -y /tmp/pipe.ts
```
Start the proxy:
```
bin/restreamer
```
Start playing:
```
cvlc http://localhost:8000/pipe.ts
```

### File Descriptors

Continuous streaming services require a lot of open file descriptors,
particularly when running a lot of low-bandwidth streams on a powerful
streaming node.

Many operating systems limit the number of file descriptors a single
process can use, so care must be taken that servers aren't starved by
this artificial limit.

As a rough guideline, determine the maximum downstream bandwidth your
server can handle, then divide by the bandwidth of each stream and
add the number of upstream connections.
Reserve some file descriptors for reconnects and fluctuations.

For example, if your server has a dedicated 10Gbit/s downstream network
connection and it serves 10 streams at 8Mbit/s, the required number
of file descriptors would be:

```
descriptor_count = streams_count + downstream_bandwidth / stream_bandwidth * 200%
=> (10 + 10 Gbit/s / 8 Mbit/s) * 2 = 2520
```

On many Linux systems, the default is very low, at 1024 file
descriptors, so restreamer would not be able to saturate all bandwidth.
With the following systemd unit file, this would be remedied:
```
[Unit]
Description=restreamer HTTP streaming proxy
After=network.target

[Service]
Type=simple
ExecStart=/usr/bin/restreamer
Restart=always
KillMode=mixed
LimitNOFILE=3000

[Install]
WantedBy=multi-user.target
```

In most cases, it's safe to set a high value, so something like
64000 (or more) would be fine.

### Memory/CPU/Network Usage

To test behaviour under heavy network load, it is recommended to run
multiple concurrent HTTP connections that only read data occasionally.

This will force restreamer to buffer excessively, increasing memory usage.

You should monitor memory usage of the restreamer process closely,
and change buffer sizes or system memory accordingly.

Note that the Go runtime does not immediately return garbage collected
memory to the operating system. This is done only about every 5 minutes
by a special reclaiming task.

Using the built-in Go profiler can be more useful to watch memory usage.
You should check out the `profiler` branch, it opens a separate web
server on port 6060 that Go pprof can connect to. See [net/http/pprof](https://golang.org/pkg/net/http/pprof/)
for instructions on its usage.

### Syscall Load

Live streaming services like restreamer require realtime or near-realtime
operation. Latency should be minimised, highlighting the importance of
small buffers. Unfortunately, small buffers will increase the number of
syscalls needed to feed data to the remote clients.

In light of the recently published [speculative execution exploits
affecting Intel CPUs](https://meltdownattack.com/), high syscall rates will
have a devastating effect on performance on these CPUs due to
mitigations in the operating system.

For this reason, restreamer is relying on buffered I/O, with default buffer
sizes as set in the Go http package. In the current implementation,
a 2KiB and a 4KiB buffer is used. This should give a good compromise
between moderate latency and limited system load.

Even with these buffers, performance loss compared with unmitigated systems
is considerable. If this is unacceptable, it is highly recommended to run
restreamer on a system that does not need/use Meltdown mitigations.
