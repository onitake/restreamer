all: server source

clean:
	rm -f server source

server: server.go client.go myerrors.go packet.go queue.go main.go
	go build -o $@ $^

source: source.go
	go build -o $@ $^

