#!/bin/bash

echo "Regenerating translation strings..."
go generate ./...

echo "Building assets"
pnpm -C ./web/assets build


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

