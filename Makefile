# enable profiling
#export GODEBUG = gctrace=1
# use go netcode instead of libc
export CGO_ENABLED = 0

# all release binaries to build by default
RELEASE_BINARIES := restreamer-linux-amd64 restreamer-linux-386 restreamer-linux-arm restreamer-linux-arm64 restreamer-darwin-amd64 restreamer-windows-amd64.exe restreamer-windows-386.exe

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

# release builds - use docker to cross-compile to various architectures

release: SHA256SUMS

SHA256SUMS: $(RELEASE_BINARIES)
	sha256sum $^ > $@

restreamer-linux-amd64:
	docker run -e GOOS=linux -e GOARCH=amd64 -e GOCACHE=/go/.cache -e CGO_ENABLED=0 -ti --rm --user $(shell id -u):$(shell id -g) -v $(shell pwd):/go/restreamer -w /go/restreamer golang:1 go build -o $@ ./cmd/restreamer

restreamer-linux-386:
	docker run -e GOOS=linux -e GOARCH=386 -e GOCACHE=/go/.cache -e CGO_ENABLED=0 -ti --rm --user $(shell id -u):$(shell id -g) -v $(shell pwd):/go/restreamer -w /go/restreamer golang:1 go build -o $@ ./cmd/restreamer

restreamer-linux-arm:
	docker run -e GOOS=linux -e GOARCH=arm -e GOCACHE=/go/.cache -e CGO_ENABLED=0 -ti --rm --user $(shell id -u):$(shell id -g) -v $(shell pwd):/go/restreamer -w /go/restreamer golang:1 go build -o $@ ./cmd/restreamer

restreamer-linux-arm64:
	docker run -e GOOS=linux -e GOARCH=arm64 -e GOCACHE=/go/.cache -e CGO_ENABLED=0 -ti --rm --user $(shell id -u):$(shell id -g) -v $(shell pwd):/go/restreamer -w /go/restreamer golang:1 go build -o $@ ./cmd/restreamer

restreamer-darwin-amd64:
	docker run -e GOOS=darwin -e GOARCH=amd64 -e GOCACHE=/go/.cache -e CGO_ENABLED=0 -ti --rm --user $(shell id -u):$(shell id -g) -v $(shell pwd):/go/restreamer -w /go/restreamer golang:1 go build -o $@ ./cmd/restreamer

# cross-compling from linux to non-amd64 darwin doesn't seem to work, even with CGO_ENABLED=0
restreamer-darwin-arm64:
	docker run -e GOOS=darwin -e GOARCH=arm64 -e GOCACHE=/go/.cache -e CGO_ENABLED=0 -ti --rm --user $(shell id -u):$(shell id -g) -v $(shell pwd):/go/restreamer -w /go/restreamer golang:1 go build -o $@ ./cmd/restreamer

restreamer-windows-amd64.exe:
	docker run -e GOOS=windows -e GOARCH=amd64 -e GOCACHE=/go/.cache -e CGO_ENABLED=0 -ti --rm --user $(shell id -u):$(shell id -g) -v $(shell pwd):/go/restreamer -w /go/restreamer golang:1 go build -o $@ ./cmd/restreamer

restreamer-windows-386.exe:
	docker run -e GOOS=windows -e GOARCH=386 -e GOCACHE=/go/.cache -e CGO_ENABLED=0 -ti --rm --user $(shell id -u):$(shell id -g) -v $(shell pwd):/go/restreamer -w /go/restreamer golang:1 go build -o $@ ./cmd/restreamer
