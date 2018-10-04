#!/bin/bash

# Copyright 2018, OpenCensus Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This file is useful to help build binaries or the Docker image
# Usage:
#   It provides a couple of commands like:
# a) Building binaries by passing in the GOOS value e.g.
#    darwin
#    linux
#    windows
#
# b) Build all the binaries by passing in: binaries
#
# c) Building the Docker image:
#    By passing in argument "docker" and optionally the version to be tagged otherwise it'll be "v0.0.1"


if [[ $# -lt 1 ]]
then
    echo -e "Usage: $0 <cmd>\nwhere <cmd> is any of 'docker' 'binaries' or GOOS such as:\ndarwin\nlinux\nwindows"
    exit
fi

function build() {
    GOOS="$1"
    LDFLAGS="\"-X github.com/census-instrumentation/opencensus-service/internal/version.GitHash=`git rev-parse --short HEAD`\""
    CMD="GOOS=$GOOS go build -ldflags $LDFLAGS -o bin/ocagent_$GOOS ./cmd/ocagent"
    echo $CMD
    eval $CMD
}

function buildAll() {
    build darwin
    build linux
    build windows
}

function dockerBuild() {
    build linux
    VERSION="$1"
    if [[ "$VERSION" == "" ]]
    then
        VERSION="v0.0.1"
    fi

    CMD="docker build -t ocagent:$VERSION ."
    echo $CMD
    eval $CMD
}

case $1 in
"docker")
    dockerBuild $2;;
"binaries")
    buildAll;;
*)
    build "$1"
esac
