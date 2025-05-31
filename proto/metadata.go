package proto

//go:generate protoc -I ../.local/protovalidate/proto/protovalidate -I ./ ./metadata.proto --go_out=./
//go:generate protoc -I ../.local/protovalidate/proto/protovalidate -I ./ ./metadata.proto --go-grpc_out=./
