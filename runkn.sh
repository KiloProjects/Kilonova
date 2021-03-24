#!/bin/bash

go build -v ./cmd/kn || exit 2

sudo ./kn main
