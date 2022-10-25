#!/bin/bash

#while true
#do
	go build -race -v ./cmd/kn || exit 2

	mv kn knnnn # fix gitignore issue

	./knnnn main
#done
