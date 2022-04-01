all: build

.PHONY: build clean test test-update doc

doc:
	go run scripts/builddoc.go > README.md

build:
	go build

test:
	go test ./...

coverage:
	go test -race -covermode=atomic -coverprofile=coverage.out ./...

test-update:
	go test ./... --update

clean:
	rm ./knut
