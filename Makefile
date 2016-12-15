export GOPATH=$(shell pwd)

all: server source

clean:
	rm -f bin/* pkg/*

server: src/main.go pkg/librestreamer.a
	go build -o $@ src/main.go

source: src/source.go
	go build -o $@ $^

pkg/librestreamer.a: src/restreamer/connection.go src/restreamer/packet.go src/restreamer/client.go src/restreamer/server.go
	go build -o $@ $^
