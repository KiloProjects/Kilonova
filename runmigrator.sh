#!/bin/bash

go build -v -tags netgo,osusergo || exit 2

./Kilonova migrateTests
