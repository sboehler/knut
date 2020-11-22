all: build

.PHONY: build clean test

build:
	go build 

test:
	go test -v ./...

clean:
	rm ./knut
