restreamer
==========

HTTP transport stream proxy

Copyright © 2016-2017 Gregor Riepl
All rights reserved.

Please see the LICENSE file for details on permitted use of this software.


Introduction
------------

restreamer is a proof-of-concept implementation of a streaming proxy
that can fetch, buffer and distribute [MPEG transport streams](https://en.wikipedia.org/wiki/MPEG-TS).
Thus, it serves as a non-transcoding streaming proxy.

In contrast to other video and audio streaming techniques (such as MPEG-DASH),
this can not easily be achieved with most HTTP proxies.
Instead of caching blocks of data, the streaming proxy needs to buffer and
distribute packets on the fly.


Architecture
------------

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
* restreamer - core program that glues the components together

Additionally, there is a simple web server that serves static files.
It not suitable for stream testing however.
For proper testing, it is recommended to use a more sophisticated
software package such as ffmpeg.


Compilation
-----------

To compile restreamer, the accompanying Makefile can be used.
You need GNU make to build it, however.
Type
```
make
```
at the command prompt, which should yield bin/restreamer .


Configuration
-------------

All configuration is done through restreamer.json .

See restreamer.example.json for a documented example.


Testing
-------

Using the example config and ffmpeg, a test stream can be set up:

Create a pipe:
```
mkfifo /tmp/pipe.ts
```
Start feeding data into the pipe (note the -re parameter):
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
