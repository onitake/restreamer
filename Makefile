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
LIB_SOURCES=api.go stats.go connection.go packet.go client.go streamer.go proxy.go acl.go config.go log.go set.go atomic.go

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
