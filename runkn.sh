#!/bin/bash

yarn --cwd ./web/assets buildJS

go build -v ./cmd/kn || exit 2

mv kn knnnn # fix gitignore issue

./knnnn main
