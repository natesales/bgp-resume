DIST_DIR=dist

all: clean test daemon client

clean:
	rm -rf $(DIST_DIR)

test:
	cd internal/encoding/ && go test

daemon:
	go build -o $(DIST_DIR)/daemon cmd/daemon/daemon.go

client:
	go build -o $(DIST_DIR)/client cmd/client/client.go
