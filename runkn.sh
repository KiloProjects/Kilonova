#!/bin/bash

go build -v ./cmd/knnnn || exit 2

sudo ./knnnn main
