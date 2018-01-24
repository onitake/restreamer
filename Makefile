export GOPATH=$(shell pwd)
# profiling
#export GODEBUG=gctrace=1
# build from pure Go, without libc
export CGO_ENABLED=0
# cross compilation
#export GOOS=windows
#export GOARCH=amd64

RESTREAMER_SOURCES=cmd/restreamer.go cmd/profile.go
LIB_SOURCES=src/restreamer/api.go src/restreamer/stats.go src/restreamer/connection.go src/restreamer/packet.go src/restreamer/client.go src/restreamer/streamer.go src/restreamer/proxy.go src/restreamer/acl.go src/restreamer/config.go src/restreamer/log.go src/restreamer/set.go src/restreamer/atomic.go

.PHONY: all clean test

all: bin/restreamer

clean:
	rm -f bin/*

docker: bin/restreamer
	docker build -t restreamer .

bin/restreamer: $(RESTREAMER_SOURCES) $(LIB_SOURCES)
	go build -o $@ $(RESTREAMER_SOURCES)
