package bw

//go:generate protoc -I=.protocol --go_out=plugins=grpc:deployment/agent .protocol/agent.proto
