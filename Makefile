version?=`if [ -d ./.git ]; then git describe --tags; else echo default; fi`

all: build

.PHONY: build clean test test-update doc

doc:
	go run scripts/builddoc.go > README.md

build: gen_version
	go build .

gen_version:
	VERSION=$(version) go run scripts/$@/main.go

test: gen_version
	go test -v ./...

test-update: gen_version
	go test ./... --update

clean:
	rm ./knut
