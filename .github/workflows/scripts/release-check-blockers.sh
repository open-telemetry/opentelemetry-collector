#!/bin/bash -ex
#
# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0

BLOCKERS=$( gh issue list --search "label:release:blocker" --json url --jq '.[].url' --repo "${REPO}" )
if [ "${BLOCKERS}" != "" ]; then
    echo "Release blockers in ${REPO} repo: ${BLOCKERS}"
    exit 1
fi
