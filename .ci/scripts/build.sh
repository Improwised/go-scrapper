#!/bin/sh -x
#
# Go scraper build with concourse caching enabled

TASK_ROOT="$(pwd)"

## Restore cache
# cp -a ${TASK_ROOT}/go/pkg/mod/* /go/pkg/mod/ || true

cd repo
go mod download

## Save cache
# cp -a /go/pkg/mod/* ${TASK_ROOT}/go/pkg/mod/ || true

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build main.go
echo -n "go-scrapper $(cat .git/ref)" >name
