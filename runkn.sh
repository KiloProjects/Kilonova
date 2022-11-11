#!/bin/bash

echo "Building js bundle"
yarn --cwd ./web/assets prodJS

go build -race -v ./cmd/kn || exit 2

mv kn knnnn # fix gitignore issue

# If it keeps crashing, restart
while true
do
	echo "Starting server..."
	sudo ./knnnn main
	echo "Server stopped..."
	sleep 2
done

