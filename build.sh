#!/usr/bin/env bash

set -o pipefail
set -x

go build http-redirector

# Build for ARM
GOOS=linux GOARCH=arm go build -o http-redirector_arm http-redirector
