#!/bin/bash -e

# Copyright Contributors to the Open Cluster Management project

# Go tools
_OS=$(go env GOOS)
_ARCH=$(go env GOARCH)

if ! which patter >/dev/null; then
    echo "Installing patter ..."
    go install github.com/apg/patter@latest
fi
if ! which gocovmerge >/dev/null; then
    echo "Installing gocovmerge..."
    go install github.com/wadey/gocovmerge@latest
fi

# Build tools

# Image tools

# Check tools
