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
* api/stats - stat collector and tracker
* streaming/proxy - static web server and proxy
* restreamer - core program that glues the components together


## Compilation

restreamer is go-gettable.
Just invoke:

```
CGO_ENABLED=0 go build github.com/onitake/restreamer
```

Disabling CGO is recommended, as this will produce a standalone binary that
does not depend on libc. This is useful for running restreamer in a bare container.

A makefile is also provided, allowing builds outside GOPATH.
Simply invoke `make` to build `bin/restreamer`.

Passing GOOS and/or GOARCH to `make` will yield cross-compiled binaries for the
respective platform. For example:

```
make GOOS=windows GOARCH=amd64
```

You can also use `make test` to run the test suite.


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


## Testing

### Test Stream

Using the example config and ffmpeg, a test stream can be set up.

Create a pipe:
```
mkfifo /tmp/pipe.ts
```
Start feeding data into the pipe (note the -re parameter for real-time streaming):
```
ffmpeg -re -i test.ts -c:a copy -c:v copy -y /tmp/pipe.ts
```
Start the proxy:
```
bin/restreamer
```
Start playing:
```
cvlc http://localhost:8000/pipe.ts
```

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
