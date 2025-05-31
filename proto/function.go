package proto

//go:generate protoc -I ../.local/protovalidate/proto/protovalidate -I ./ ./function.proto --go_out=./
//go:generate protoc -I ../.local/protovalidate/proto/protovalidate -I ./ ./function.proto --go-grpc_out=./
