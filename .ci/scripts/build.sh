#!/bin/sh -x
#
# Go scraper build with concourse caching enabled

## For caching
GOPATH=${PWD}/go-cache
export GOPATH

## Change dir
cd repo

## go go go
go mod download
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build main.go

## for github-release-resource > name
echo -n "go-scrapper $(cat .git/ref)" >name
