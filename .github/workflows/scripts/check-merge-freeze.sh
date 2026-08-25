#!/bin/bash -e
#
# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0

# Check for [chore] Prepare release PRs in core repo
BLOCKERS=$( gh pr list -A "otelbot[bot]" -S "[chore] Prepare release" --json url -q '.[].url' -R "${REPO}" )

# Check for [chore] Update core dependencies PRs in opentelemetry-collector-contrib
CONTRIB_REPO="open-telemetry/opentelemetry-collector-contrib"
CONTRIB_BLOCKERS=$( gh pr list -A "otelbot[bot]" -S "[chore] Update core dependencies" --json url -q '.[].url' -R "${CONTRIB_REPO}" )

# Combine both blockers
BLOCKERS="${BLOCKERS}${BLOCKERS:+ }${CONTRIB_BLOCKERS}"

if [ "${BLOCKERS}" != "" ]; then
    echo "Merging in main is frozen, as there are open release/update PRs: ${BLOCKERS}"
    echo "If you believe this is no longer true, re-run this job to unblock your PR."
    exit 1
fi
