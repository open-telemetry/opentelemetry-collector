#!/bin/bash -e
#
# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0

BLOCKERS=$( gh pr list --search "label:release:prepare" --json url --jq '.[].url' --repo "${REPO}" )
if [ "${BLOCKERS}" != "" ]; then
    echo "Merging in main is frozen while release PR is open: ${BLOCKERS}"
    exit 1
fi
