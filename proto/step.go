package proto

//go:generate protoc -I ../.local/protovalidate/proto/protovalidate -I ./ ./step.proto --go_out=./
//go:generate protoc -I ../.local/protovalidate/proto/protovalidate -I ./ ./step.proto --go-grpc_out=./
