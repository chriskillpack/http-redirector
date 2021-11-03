#!/usr/bin/env bash

set -o pipefail
set -x

go build http-redirector
if [ $? -ne 0 ]
then
    exit
fi

# Build for ARM
GOOS=linux GOARCH=arm go build -o http-redirector_arm http-redirector
