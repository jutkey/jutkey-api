#!/bin/bash
set -e -x

HOMEDIR=$(pwd)

function buildpkg() {
    buildBin=$1
    buildModule=$2
    buildFile=$3
    buildBranch=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo unknown)
    buildDate=$(date -u "+%Y-%m-%d-%H:%M:%S(UTC)")
    commitHash=$(git rev-parse --short HEAD 2>/dev/null || echo unknown)
    go build -o "$buildBin" -ldflags "-s -w -X $buildModule/packages/consts.buildBranch=$buildBranch -X $buildModule/packages/consts.buildDate=$buildDate -X $buildModule/packages/consts.commitHash=$commitHash" "$buildFile"
}

buildpkg jutkey-server "jutkey-server" "$HOMEDIR/main.go"
