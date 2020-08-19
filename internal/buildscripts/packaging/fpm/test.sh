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

set -euo pipefail

SCRIPT_DIR="$( cd "$( dirname ${BASH_SOURCE[0]} )" && pwd )"
REPO_DIR="$( cd "$SCRIPT_DIR/../../../../" && pwd )"
PKG_PATH=${1:-}

. $SCRIPT_DIR/common.sh

if [[ -z "$PKG_PATH" ]]; then
    echo "usage: ${BASH_SOURCE[0]} DEB_OR_RPM_PATH" >&2
    exit 1
fi

if [[ ! -f "$PKG_PATH" ]]; then
    echo "$PKG_PATH not found!" >&2
    exit 1
fi

pkg_name="$( basename "$PKG_PATH" )"
pkg_type="${pkg_name##*.}"
if [[ ! "$pkg_type" =~ ^(deb|rpm)$ ]]; then
    echo "$PKG_PATH not supported!" >&2
    exit 1
fi
image_name="otelcol-$pkg_type-test"
container_name="$image_name"
docker_run="docker run --name $container_name -d -v /sys/fs/cgroup:/sys/fs/cgroup:ro --privileged $image_name"
docker_exec="docker exec $container_name"

trap "docker rm -fv $container_name >/dev/null 2>&1 || true" EXIT

docker build -t $image_name -f "$SCRIPT_DIR/$pkg_type/Dockerfile.test" "$SCRIPT_DIR"
docker rm -fv $container_name >/dev/null 2>&1 || true

# test install
echo
$docker_run
install_pkg "$PKG_PATH"

# ensure service has started and still running after 5 seconds
sleep 5
echo "Checking $SERVICE_NAME service status ..."
$docker_exec systemctl --no-pager status $SERVICE_NAME

echo "Checking $PROCESS_NAME process ..."
$docker_exec pgrep -a -u otel $PROCESS_NAME

# test uninstall
echo
uninstall_pkg $pkg_type

echo "Checking $SERVICE_NAME service status after uninstall ..."
if $docker_exec systemctl --no-pager status $SERVICE_NAME; then
    echo "$SERVICE_NAME service still running after uninstall!" >&2
    exit 1
fi
echo "$SERVICE_NAME service successfully stopped after uninstall"

echo "Checking $SERVICE_NAME service existence after uninstall ..."
if $docker_exec systemctl list-unit-files --all | grep $SERVICE_NAME; then
    echo "$SERVICE_NAME service still exists after uninstall!" >&2
    exit 1
fi
echo "$SERVICE_NAME service successfully removed after uninstall"

echo "Checking $PROCESS_NAME process after uninstall ..."
if $docker_exec pgrep $PROCESS_NAME; then
    echo "$PROCESS_NAME process still running after uninstall!" >&2
    exit 1
fi
echo "$PROCESS_NAME process successfully killed after uninstall"
