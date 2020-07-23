#!/bin/bash

set -euxo pipefail

GOOS="${GOOS:-linux}"
GOARCH="${GOARCH:-amd64}"
CIRCLE_JOB="${CIRCLE_JOB:-goweight-otelcol_${GOOS}_${GOARCH}}"
OTELCOL_BIN="otelcol_${GOOS}_${GOARCH}"
if [[ "$GOOS" = "windows" ]]; then
    OTELCOL_BIN="${OTELCOL_BIN}.exe"
fi

mkdir -p ~/goweight
goweight --json ./cmd/otelcol | jq . > ${CIRCLE_JOB}.json

# add otelcol binary size to goweight results
otelcol_size=$( du -b bin/${OTELCOL_BIN} | cut -f1 )
otelcol_size_human=$( du -h bin/${OTELCOL_BIN} | cut -f1 )
cat ${CIRCLE_JOB}.json | jq ". + [{\"path\": \"${OTELCOL_BIN}\", \"name\": \"${OTELCOL_BIN}\", \"size\": $otelcol_size, \"size_human\": \"$otelcol_size_human\"}]" | tee ~/goweight/${CIRCLE_JOB}.json
