#!/bin/bash


protoc --proto_path=rpc --go_out=grpc --go-grpc_out=grpc --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative rpc/*/*.proto
