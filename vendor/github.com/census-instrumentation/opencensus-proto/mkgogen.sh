#!/usr/bin/env bash

# Run this if opencensus-proto is checked in the GOPATH.
# go get -d github.com/census-instrumentation/opencensus-proto
# to check in the repo to the GOAPTH.
#
# To generate:
#
# cd $(go env GOPATH)/census-instrumentation/opencensus-proto
# ./mkgogen.sh

OUTDIR="$(go env GOPATH)/src"

protoc --go_out=plugins=grpc:$OUTDIR opencensus/proto/stats/stats.proto \
    && protoc --go_out=plugins=grpc:$OUTDIR opencensus/proto/stats/metrics/metrics.proto \
    && protoc --go_out=plugins=grpc:$OUTDIR opencensus/proto/trace/trace.proto \
    && protoc -I=. --go_out=plugins=grpc:$OUTDIR opencensus/proto/exporter/exporter.proto
