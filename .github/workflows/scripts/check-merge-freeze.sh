#!/bin/bash -e
#
# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0

BLOCKERS=$( gh pr list -A opentelemetrybot -S "[chore] Prepare release" --json url -q '.[].url' -R "${REPO}" )
if [ "${BLOCKERS}" != "" ]; then
    echo "Merging in main is frozen, as there are open \"Prepare release\" PRs: ${BLOCKERS}"
    echo "If you believe this is no longer true, re-run this job to unblock your PR."
    exit 1
fi
