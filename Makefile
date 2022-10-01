all: build

.PHONY: clean test test-update doc web

doc:
	go run scripts/builddoc.go > README.md

go_proto_out=server/proto
web_proto_out=web/src/proto

protogen = \
	$(go_proto_out)/service_grpc.pb.go \
	$(go_proto_out)/service.pb.go \
	$(web_proto_out)/service_pb.d.ts \
	$(web_proto_out)/service_pb.js \
	$(web_proto_out)/ServiceServiceClientPb.ts
	

$(protogen): proto/service.proto 
	protoc \
	-I=proto \
	--go_opt=paths=source_relative \
	--go_out=$(go_proto_out) \
	--go-grpc_opt=paths=source_relative \
	--go-grpc_out=$(go_proto_out) \
	--grpc-web_out=import_style=typescript,mode=grpcweb:$(web_proto_out) \
	--js_out=import_style=commonjs,binary:$(web_proto_out) \
	service.proto

proto: $(protogen)

web/build: $(shell find web/src web/public -type f) web/node_modules proto
	cd web && npm run build

web/node_modules web/package-lock.json: web/package.json
	cd web && npm install

test:
	go test ./...

coverage:
	go test -race -covermode=atomic -coverprofile=coverage.out ./...

test-update:
	go test ./... --update

clean:
	rm -rf web/node_modules web/build
	rm -f ./knut

knut: $(protogen) web/build
	go build

fe: proto web/node_modules
	cd web && BROWSER=none npm start

be: proto
	go run main.go web