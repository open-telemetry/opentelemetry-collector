#!/bin/bash

# Copyright The OpenTelemetry Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#       http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -euxo pipefail

SCRIPT_DIR="$( cd "$( dirname ${BASH_SOURCE[0]} )" && pwd )"
REPO_DIR="$( cd "$SCRIPT_DIR/../../../../../" && pwd )"
VERSION="${1:-}"
ARCH="${2:-"amd64"}"
OUTPUT_DIR="${3:-"$REPO_DIR/dist/"}"
OTELCOL_PATH="$REPO_DIR/bin/otelcol_linux_$ARCH"
CONFIG_PATH="$REPO_DIR/examples/local/otel-config.yaml"

mkdir -p $OUTPUT_DIR

. $SCRIPT_DIR/../common.sh

if [[ -z "$VERSION" ]]; then
    latest_tag="$( git describe --abbrev=0 --match v[0-9]* )"
    VERSION="${latest_tag}-post"
fi

fpm -s dir -t deb -n $PKG_NAME -v ${VERSION#v} -f -p "$OUTPUT_DIR" \
    --vendor "$PKG_VENDOR" \
    --maintainer "$PKG_MAINTAINER" \
    --description "$PKG_DESCRIPTION" \
    --license "$PKG_LICENSE" \
    --url "$PKG_URL" \
    --architecture "$ARCH" \
    --config-files /etc/otel-collector/config.yaml \
    --deb-dist "stable" \
    --deb-user "$PKG_USER" \
    --deb-group "$PKG_GROUP" \
    --before-install "$PREINSTALL_PATH" \
    --after-install "$POSTINSTALL_PATH" \
    --pre-uninstall "$PREUNINSTALL_PATH" \
    $OTELCOL_PATH=/usr/bin/$PROCESS_NAME \
    $SERVICE_PATH=/lib/systemd/system/$SERVICE_NAME.service \
    $CONFIG_PATH=/etc/otel-collector/config.yaml
