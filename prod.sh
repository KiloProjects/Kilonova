#!/bin/bash

go build -v || exit 2

./Kilonova -data="/data/kilonova" -config="/data/config/config.json"

