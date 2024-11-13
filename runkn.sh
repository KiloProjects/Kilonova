#!/bin/bash

echo "Regenerating translation strings..."
go generate

if [[ $* == *--css* ]]; then
    echo "Regenerating CSS"
    yarn --cwd ./web/assets prodCSS
fi

echo "Building js bundle"
yarn --cwd ./web/assets prodJS

echo "Vendoring js dependencies"
yarn --cwd ./web/assets vendor


#go build -race -v ./cmd/kn || exit 2
go build -v ./cmd/kn || exit 2

# If it keeps crashing, restart
while true
do
	echo "Starting server..."
    # Preserve overrides flag
	sudo KN_FLAG_OVERRIDES=$KN_FLAG_OVERRIDES ./kn main
	echo "Server stopped..."
	sleep 2
done

