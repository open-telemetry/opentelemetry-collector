#!/bin/bash

set -euxo pipefail

SCRIPT_DIR="$( cd "$( dirname ${BASH_SOURCE[0]} )" && pwd )"
REPO_DIR="$( cd "$SCRIPT_DIR/../../../../../" && pwd )"
VERSION="${1:-}"
ARCH="${2:-"amd64"}"
OUTPUT_DIR="${3:-"$REPO_DIR/bin/"}"
OTELCOL_PATH="$REPO_DIR/bin/otelcol_linux_amd64"
CONFIG_PATH="$REPO_DIR/examples/otel-local-config.yaml"

if [[ -z "$VERSION" ]]; then
    latest_tag="$( git describe --abbrev=0 --match v[0-9]* )"
    VERSION="${latest_tag}~post"
fi

fpm -s dir -t rpm -n otel-collector -v ${VERSION#v} -f -p "$OUTPUT_DIR" \
    --vendor "OpenTelemetry Community" \
    --maintainer "OpenTelemetry Community <cncf-opentelemetry-community@lists.cncf.io>" \
    --description "OpenTelemetry Collector" \
    --license "Apache 2.0" \
    --rpm-summary "OpenTelemetry Collector" \
    --url "https://github.com/open-telemetry/opentelemetry-collector" \
    --architecture "$ARCH" \
    --config-files /etc/otel-collector/config.yaml \
    $OTELCOL_PATH=/usr/bin/otelcol \
    $CONFIG_PATH=/etc/otel-collector/config.yaml