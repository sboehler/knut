all: build

.PHONY: build clean test doc

doc:
	go run scripts/builddoc.go > README.md

build:
	go build

test:
	go test -v ./...

clean:
	rm ./knut
