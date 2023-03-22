#!/bin/bash
set -e

if ! command -v gobadge &>/dev/null; then
    export PATH="$(go env GOPATH)/bin:$PATH"
    if ! command -v gobadge &>/dev/null; then
        go install github.com/AlexBeauchemin/gobadge@latest
    fi
fi

go test -covermode=atomic -coverprofile=coverage.out -v -p=1 -count=1 -race ./...
go tool cover -html=coverage.out -o=coverage.html
go tool cover -func=coverage.out -o=coverage.out
gobadge -filename=coverage.out -green=80 -yellow=50
