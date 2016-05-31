all: server source

clean:
	rm -f server source

server: server.go
	go build -o $@ $^

source: source.go
	go build -o $@ $^

