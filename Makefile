# profiling
#export GODEBUG=gctrace=1
# build from pure Go, without libc
export CGO_ENABLED=0
# cross compilation
#export GOOS=windows
#export GOARCH=amd64

.PHONY: all clean run

all: bin/restreamer

clean:
	rm -f bin/* pkg/*

docker: bin/restreamer
	docker build -t restreamer .

bin/restreamer: lib/api.go lib/stats.go lib/connection.go lib/packet.go lib/client.go lib/streamer.go lib/proxy.go lib/acl.go lib/config.go lib/log.go lib/set.go lib/atomic.go main.go profile.go
	go build -o $@ .
