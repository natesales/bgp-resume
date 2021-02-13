DIST_DIR=dist

all: clean build-daemon build-client

clean:
	rm -rf $(DIST_DIR)

build-daemon:
	go build -o $(DIST_DIR)/daemon cmd/daemon/daemon.go

build-client:
	go build -o $(DIST_DIR)/client cmd/client/client.go
