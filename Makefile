# enable profiling
#export GODEBUG = gctrace=1
# use go netcode instead of libc
export CGO_ENABLED = 0

# always force a rebuild of the main binary
.PHONY: all clean test fmt vendor docker restreamer

all: restreamer

clean:
	rm -rf restreamer

test:
	go test ./...

fmt:
	go fmt ./...

docker:
	docker build -t onitake/restreamer .

restreamer:
	go build ./cmd/restreamer
