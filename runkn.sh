#!/bin/bash

go build -v ./cmd/kn || exit 2

mv kn knnnn # fix gitignore issue

sudo ./knnnn main
