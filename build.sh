#!/bin/bash

# clean old builds
rm -rf dist
mkdir -p dist

# version tag (optional: pass as argument)
VERSION=${1:-"dev"}

echo "building sugarSplit $VERSION"

# linux
GOOS=linux GOARCH=amd64 go build -o dist/sugarSplit-linux-amd64 ./cmd/sugarSplit
GOOS=linux GOARCH=arm64 go build -o dist/sugarSplit-linux-arm64 ./cmd/sugarSplit

# macos
GOOS=darwin GOARCH=amd64 go build -o dist/sugarSplit-macos-amd64 ./cmd/sugarSplit
GOOS=darwin GOARCH=arm64 go build -o dist/sugarSplit-macos-arm64 ./cmd/sugarSplit

echo "done! binaries in dist/"
ls -lh dist/
