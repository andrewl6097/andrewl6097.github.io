#!/bin/bash

set -e
set -x

GOOS=linux GOARCH=amd64 go build -o main main.go

zip main.zip main
