export GOPATH=$(shell pwd)

all: bin/server bin/source

clean:
	rm -f bin/* pkg/*

bin/server: src/main.go pkg/librestreamer.a
	go build -o $@ src/main.go

bin/source: src/source.go
	go build -o $@ $^

pkg/librestreamer.a: src/restreamer/connection.go src/restreamer/packet.go src/restreamer/client.go src/restreamer/server.go
	go build -o $@ $^
