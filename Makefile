version?=`if [ -d ./.git ]; then git describe --tags; else echo default; fi`
date    =`date "+%Y-%m-%d"`
package ="github.com/sboehler/knut/cmd"
ldflags ="-X $(package).version=$(version) -X $(package).date=$(date)"

all: build

.PHONY: build clean test test-update doc

doc:
	go run scripts/builddoc.go > README.md

build:
	go build -ldflags $(ldflags)

test:
	go test -v ./...

test-update:
	go test ./... --update

clean:
	rm ./knut
