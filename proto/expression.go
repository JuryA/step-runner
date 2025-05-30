package proto

//go:generate protoc -I ../.local/protovalidate/proto/protovalidate -I ./ ./expression.proto --go_out=./
//go:generate protoc -I ../.local/protovalidate/proto/protovalidate -I ./ ./expression.proto --go-grpc_out=./
