//go:generate protoc --go_opt=paths=source_relative --go_out=. --go-grpc_out=. --go-grpc_opt=paths=source_relative service.proto  --grpc-web_out=import_style=typescript,mode=grpcweb:../../web/src/proto --js_out=import_style=commonjs,binary:../../web/src/proto

package service_go_proto
