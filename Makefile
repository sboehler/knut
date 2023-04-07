all: knut

.PHONY: clean test test-update doc install

doc:
	go run scripts/builddoc.go > README.md

test:
	go test ./...

coverage:
	go test -race -covermode=atomic -coverprofile=coverage.out ./...

test-update:
	go test ./... --update

clean:
	rm -f ./knut

knut:
	go build

install:
	go install