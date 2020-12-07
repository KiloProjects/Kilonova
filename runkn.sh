#!/bin/bash

# konsole -e redis-server

# go run -v -tags netgo,osusergo ./cmd/kilonova -data="/home/alexv/src/kninfo/data" -debug=true -logDir="/home/alexv/src/knlogs"

go build -v -tags netgo,osusergo || exit 2

./Kilonova main
# ./Kilonova -dataDir="/home/alexv/src/kninfo/data" -debug=true main

# ./Kilonova -data="/home/alexv/src/kninfo/data" -debug=true -logDir="/home/alexv/src/kninfo/logs"
