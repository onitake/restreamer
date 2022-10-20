# enable profiling
#export GODEBUG = gctrace=1
# use go netcode instead of libc
export CGO_ENABLED = 0
# set the build container Go version
GO_VERSION = 1.19

# all release binaries to build by default
RELEASE_BINARIES := restreamer-linux-amd64 restreamer-linux-386 restreamer-linux-arm restreamer-linux-arm64 restreamer-darwin-amd64 restreamer-darwin-arm64 restreamer-windows-amd64.exe restreamer-windows-386.exe  restreamer-windows-arm64.exe

# always force a rebuild of the main binary
.PHONY: all clean test fmt vendor docker restreamer

all: restreamer

clean:
	rm -rf restreamer ${RELEASE_BINARIES}

test:
	go vet ./...
	go test ./...

fmt:
	go fmt ./...

docker:
	podman build -t onitake/restreamer .

restreamer:
	go build ./cmd/restreamer

# release builds - use podman to cross-compile to various architectures

release: SHA256SUMS

SHA256SUMS: $(RELEASE_BINARIES)
	sha256sum $^ > $@

restreamer-linux-amd64:
	podman run -e GOOS=linux -e GOARCH=amd64 -e GOCACHE=/go/.cache -e CGO_ENABLED=0 --rm -v $(shell pwd):/go/restreamer -w /go/restreamer golang:${GO_VERSION} go build -o $@ ./cmd/restreamer

restreamer-linux-386:
	podman run -e GOOS=linux -e GOARCH=386 -e GOCACHE=/go/.cache -e CGO_ENABLED=0 --rm -v $(shell pwd):/go/restreamer -w /go/restreamer golang:${GO_VERSION} go build -o $@ ./cmd/restreamer

restreamer-linux-arm:
	podman run -e GOOS=linux -e GOARCH=arm -e GOCACHE=/go/.cache -e CGO_ENABLED=0 --rm -v $(shell pwd):/go/restreamer -w /go/restreamer golang:${GO_VERSION} go build -o $@ ./cmd/restreamer

restreamer-linux-arm64:
	podman run -e GOOS=linux -e GOARCH=arm64 -e GOCACHE=/go/.cache -e CGO_ENABLED=0 --rm -v $(shell pwd):/go/restreamer -w /go/restreamer golang:${GO_VERSION} go build -o $@ ./cmd/restreamer

restreamer-darwin-amd64:
	podman run -e GOOS=darwin -e GOARCH=amd64 -e GOCACHE=/go/.cache -e CGO_ENABLED=0 --rm -v $(shell pwd):/go/restreamer -w /go/restreamer golang:${GO_VERSION} go build -o $@ ./cmd/restreamer

restreamer-darwin-arm64:
	podman run -e GOOS=darwin -e GOARCH=arm64 -e GOCACHE=/go/.cache -e CGO_ENABLED=0 --rm -v $(shell pwd):/go/restreamer -w /go/restreamer golang:${GO_VERSION} go build -o $@ ./cmd/restreamer

restreamer-windows-amd64.exe:
	podman run -e GOOS=windows -e GOARCH=amd64 -e GOCACHE=/go/.cache -e CGO_ENABLED=0 --rm -v $(shell pwd):/go/restreamer -w /go/restreamer golang:${GO_VERSION} go build -o $@ ./cmd/restreamer

restreamer-windows-386.exe:
	podman run -e GOOS=windows -e GOARCH=386 -e GOCACHE=/go/.cache -e CGO_ENABLED=0 --rm -v $(shell pwd):/go/restreamer -w /go/restreamer golang:${GO_VERSION} go build -o $@ ./cmd/restreamer

restreamer-windows-arm64.exe:
	podman run -e GOOS=windows -e GOARCH=arm64 -e GOCACHE=/go/.cache -e CGO_ENABLED=0 --rm -v $(shell pwd):/go/restreamer -w /go/restreamer golang:${GO_VERSION} go build -o $@ ./cmd/restreamer
