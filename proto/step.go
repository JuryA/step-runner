package proto

//go:generate protoc -I ./ ./step.proto --go_out=./
//go:generate protoc -I ./ ./step.proto --go-grpc_out=./
