version?=`if [ -d ./.git ]; then git describe --tags; else echo default; fi`

all: build

.PHONY: build clean test test-update doc

doc:
	go run scripts/builddoc.go > README.md

build: gen_version
	go build .

gen_version:
	VERSION=$(version) go run scripts/$@.go

test:
	go test -v ./...

test-update:
	go test ./... --update

clean:
	rm ./knut
