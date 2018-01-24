export GOPATH=$(shell pwd)
# profiling
#export GODEBUG=gctrace=1
# build from pure Go, without libc
export CGO_ENABLED=0
# cross compilation
#export GOOS=windows
#export GOARCH=amd64

PACKAGE_PREFIX=github.com/onitake
PACKAGE=restreamer
PACKAGE_PATH=src/$(PACKAGE_PREFIX)/$(PACKAGE)
RESTREAMER_SOURCES=cmd/restreamer.go cmd/profile.go
UTIL_SOURCES=util/log.go util/set.go util/atomic.go util/shuffle.go
API_SOURCES=api/api.go api/stats.go
MPEGTS_SOURCES=mpegts/packet.go
STREAMING_SOURCES=streaming/connection.go streaming/client.go streaming/streamer.go streaming/proxy.go streaming/acl.go streaming/config.go streaming/manager.go
LIB_SOURCES=$(UTIL_SOURCES) $(API_SOURCES) $(MPEGTS_SOURCES) $(STREAMING_SOURCES)

.PHONY: all clean test

all: bin/restreamer

$(PACKAGE_PATH):
	mkdir -p "src/$(PACKAGE_PREFIX)"
	ln -s "$(shell pwd)" "$(PACKAGE_PATH)"

clean:
	rm -f bin/* $(PACKAGE_PATH)
	rm -rf src

docker: bin/restreamer
	docker build -t restreamer .

bin/restreamer: $(PACKAGE_PATH) $(RESTREAMER_SOURCES) $(LIB_SOURCES)
	go build -o $@ $(RESTREAMER_SOURCES)
