#!/usr/bin/env sh
if [[ "$1" == "" ]]; then
    VERSION=""
else
    VERSION="-$1"
fi

rm -f rump-*
go build -o rump$VERSION ../cmd/rump/main.go
