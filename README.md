# restreamer

HTTP transport stream proxy

Copyright © 2016-2017 Gregor Riepl
All rights reserved.

Please see the LICENSE file for details on permitted use of this software.


## Introduction

restreamer is a proof-of-concept implementation of a streaming proxy
that can fetch, buffer and distribute [MPEG transport streams](https://en.wikipedia.org/wiki/MPEG-TS).
Thus, it serves as a non-transcoding streaming proxy.

In contrast to other video and audio streaming techniques (such as MPEG-DASH),
this can not easily be achieved with most HTTP proxies.
Instead of caching blocks of data, the streaming proxy needs to buffer and
distribute packets on the fly.


## Architecture

restreamer is written in [Go](https://golang.org/).
Go offers many primitives and an extensive standard library that contains
most building blocks for a web server. This makes it a first-rate
choice for such a product.

There are several key components:
* Client - HTTP getter for fetching upstream data
* Connection - HTTP server that feeds data to clients
* Streamer - connection broker and data queue
* Api - web API for service monitoring
* Statistics - stat collector and tracker
* Proxy - static web server and proxy
* restreamer - core program that glues the components together

Additionally, there is a simple web server that serves static files.
It not suitable for stream testing however.
For proper testing, it is recommended to use a more sophisticated
software package such as ffmpeg.


## Compilation

To compile restreamer, the accompanying Makefile can be used.
You need GNU make to build it, however.
Type
```
make
```
at the command prompt, which should yield bin/restreamer .


## Configuration

All configuration is done through restreamer.json .

See restreamer.example.json for a documented example.

The input and output buffer sizes should be adapted to the expected
stream bitrates, to allow for a certain amount of resilience.

Memory usage is primarily tied to these values, and can be (roughly)
calculated as follows:

```
mpegts_packet_size = 188
max_buffer_memory = mpegts_packet_size * (input_buffer_size + max_connections * output_buffer_size)
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
