# restreamer

HTTP transport stream proxy

Copyright © 2016-2017 Gregor Riepl;
All rights reserved.

Please see the LICENSE file for details on permitted use of this software.


## Introduction

restreamer is a proof-of-concept implementation of a streaming proxy
that can fetch, buffer and distribute [MPEG transport streams](https://en.wikipedia.org/wiki/MPEG-TS).
It serves as a non-transcoding streaming proxy for legacy HbbTV applications.

Data sources can be local files, remote HTTP servers or raw TCP streams.


## Architecture

restreamer is written in [Go](https://golang.org/), a very versatile
programming language, and suited perfectly for the development of
network services.

These are the key components:
* Client - HTTP getter for fetching upstream data
* Connection - HTTP server that feeds data to clients
* Streamer - connection broker and data queue
* Api - web API for service monitoring
* Statistics - stat collector and tracker
* Proxy - static web server and proxy
* restreamer - core program that glues the components together


## Compilation

Compiling restreamer is very easy if you have GNU make installed.
Just run `make` to build `bin/restreamer`.

It is also possible to add the source code repository to your GOPATH
and build restreamer manually.


## Configuration

All configuration is done through a configuration file named `restreamer.json`.

See restreamer.example.json for a documented example.

The input and output buffer sizes should be adapted to the expected
stream bitrates and account for unstable or slow internet connections.

It is also important to keep the bandwidth of the network interfaces
in mind, so the connection limit should be set accordingly.

Memory usage is also tied to these values, and can be roughly calculated as follows:

```
mpegts_packet_size = 188
max_buffer_memory = mpegts_packet_size * (number_of_streams * input_buffer_size + max_connections * output_buffer_size)
```


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

### Memory Usage

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
