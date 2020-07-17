#!/usr/bin/env bash

# build the binary
GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o linear-autolabeler

# build the docker image
docker build --tag linear-autolabeler:0.1 .
