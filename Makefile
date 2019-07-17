# enable profiling
#export GODEBUG = gctrace=1
# use go netcode instead of libc
export CGO_ENABLED = 0
# enforce using gomod
export GO111MODULE = on

PACKAGE:=github.com/onitake/restreamer

# always force a rebuild of the main binary
.PHONY: all clean test fmt vendor docker restreamer

all: restreamer

clean:
	rm -rf restreamer

test:
	go test $(PACKAGE)/...

fmt:
	go fmt $(PACKAGE)/...

vendor:
	go mod tidy
	go mod vendor

docker: restreamer
	docker build -t restreamer .

restreamer:
	go build ./cmd/restreamer
