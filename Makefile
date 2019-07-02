# override the go path to allow building outside workspace
export GOPATH = $(shell pwd)
# enable profiling
#export GODEBUG = gctrace=1
# use go netcode instead of libc
export CGO_ENABLED = 0

# cross compilation
# example:
# make GOOS=windows GOARCH=amd64
# will produce bin/restreamer-windows-amd64.exe
ifdef GOOS
export GOOS = $(GOOS)
PACKAGE_OS := -$(GOOS)
endif
ifdef GOARCH
export GOARCH = $(GOARCH)
PACKAGE_ARCH += -$(GOARCH)
endif
ifeq ($(OS),Windows_NT)
	EXE_SUFFIX = .exe
else ifeq ($(GOOS),windows)
	EXE_SUFFIX = .exe
else
	EXE_SUFFIX =
endif

PACKAGE_PREFIX=github.com/onitake
PACKAGE=restreamer
PACKAGE_PATH=src/$(PACKAGE_PREFIX)/$(PACKAGE)
RESTREAMER_SOURCES=restreamer.go profile.go logger.go
RESTREAMER_EXE=restreamer$(PACKAGE_OS)$(PACKAGE_ARCH)$(EXE_SUFFIX)

# always force a rebuild of the main binary
.PHONY: all clean test docker bin/$(RESTREAMER_EXE)

all: bin/$(RESTREAMER_EXE)

$(PACKAGE_PATH):
	mkdir -p "src/$(PACKAGE_PREFIX)"
	ln -s "$(shell pwd)" "$(PACKAGE_PATH)"

clean:
	rm -f bin/* $(PACKAGE_PATH)
	rm -rf src

test: $(PACKAGE_PATH)
	go test $(PACKAGE_PREFIX)/$(PACKAGE)/util \
		$(PACKAGE_PREFIX)/$(PACKAGE)/streaming \
		$(PACKAGE_PREFIX)/$(PACKAGE)/api \
		$(PACKAGE_PREFIX)/$(PACKAGE)/protocol \
		$(PACKAGE_PREFIX)/$(PACKAGE)/event \
		$(PACKAGE_PREFIX)/$(PACKAGE)/configuration \
		$(PACKAGE_PREFIX)/$(PACKAGE)/auth

docker: bin/restreamer
	docker build -t restreamer .

bin/$(RESTREAMER_EXE): $(PACKAGE_PATH) $(RESTREAMER_SOURCES)
	go build -o $@ $(RESTREAMER_SOURCES)
