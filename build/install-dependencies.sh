#!/bin/bash -e

# Copyright Contributors to the Open Cluster Management project

# Go tools
_OS=$(go env GOOS)
_ARCH=$(go env GOARCH)

if ! which patter > /dev/null;     then echo "Installing patter ..."; pushd $(mktemp -d) && GO111MODULE=off go get -u github.com/apg/patter && popd; fi
if ! which gocovmerge > /dev/null; then echo "Installing gocovmerge..."; pushd $(mktemp -d) && GO111MODULE=off go get -u github.com/wadey/gocovmerge && popd; fi

# Build tools

# Image tools

# Check tools
