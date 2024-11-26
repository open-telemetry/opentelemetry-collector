#!/bin/bash -e
#
# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0

BLOCKERS=$( gh pr list --search "label:release:merge-freeze" --json url --jq '.[].url' --repo "${REPO}" )
if [ "${BLOCKERS}" != "" ]; then
    echo "Merging in main is frozen, as there are open PRs labeled 'release:merge-freeze': ${BLOCKERS}"
    exit 1
fi
