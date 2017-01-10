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
this can not easily be achieved with most HTTP proxy packages.
At the very least, awareness of the continuous nature of such streams
and the underlying data structure is required.


Architecture
------------

restreamer is written in Go, as a bit of an experiment to get familiar with
the language. Go offers many primitives and an extensive standard library
that facilitate the development of a web server. This makes it a first-rate
choice for such a project.

There are several key components:
* Client - an implementation of an HTTP client that is capable
  of fetching data from the upstream server and feeding it into a queue
* Connection - an abstraction of a downstream client connection
* Streamer - a connection broker and data dispatcher
* server - the core program that glues the components together
* source - simple HTTP file server (for testing, better use ffmpeg)

Additionally, there is a packet source tool that acts as a simple
web server, serving data from a file to the streaming proxy.
Note however, that the stream is not rate-limited and will certainly
overflow and downstream clients. But it can be used to the server's
robustness and performance.

For proper testing, it is recommended to use a more sophisticated
software package instead, such as ffmpeg.


Compilation
-----------

To compile restreamer, the accompanying Makefile can be used.
You need GNU make to build it, however.
Type
```
make
```
at the command prompt, which should yield bin/server and bin/source .


Configuration
-------------

All configuration is done through server.json .

See the included example for its structure.


Testing
-------

Create a pipe, as configured in the example server.json:
```
mkfifo /tmp/pipe.ts
```
Start feeding data into the pipe, in realtime:
```
ffmpeg -re -i test.ts -c:a copy -c:v copy -y /tmp/pipe.ts
```
Start the proxy:
```
bin/server
```
Start playing:
```
mpv http://localhost:8000/pipe.ts
```
