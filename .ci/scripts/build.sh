#!/bin/bash -ex
#
# Go scraper build with concourse caching enabled
TASK_ROOT=$(pwd)

## For caching
export GOPATH=${TASK_ROOT}/go-cache
export PATH=${TASK_ROOT}/go-cache/bin:$PATH

cd repo

## go mod and build
go mod download
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build main.go

echo -n "go-scrapper $(cat .git/ref)" >name
echo -n "sha256sum main" >body
