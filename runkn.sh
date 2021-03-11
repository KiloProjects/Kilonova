#!/bin/bash

go build -v -tags netgo,osusergo ./cmd/kn || exit 2

sudo ./kn main
