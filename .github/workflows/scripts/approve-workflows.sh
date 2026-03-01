#!/usr/bin/env bash
#
# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0
#

set -euo pipefail

if [[ -z "${PR_NUMBER:-}" || -z "${COMMENT:-}" || -z "${SENDER:-}" || -z "${ORG_TOKEN:-}" ]]; then
    echo "PR_NUMBER, COMMENT, SENDER or ORG_TOKEN not set"
    exit 0
fi

if [[ ${COMMENT:0:18} != "/workflow-approve" ]]; then
    echo "Not a workflow-approve command"
    exit 0
fi

TEAMS=(
    "collector-triagers"
    "collector-approvers"
    "collector-maintainers"
)

IS_AUTHORIZED="false"
for TEAM in "${TEAMS[@]}"; do
    if GH_TOKEN="${ORG_TOKEN}" gh api "orgs/open-telemetry/teams/${TEAM}/memberships/${SENDER}" --silent 2>/dev/null; then
        IS_AUTHORIZED="true"
        break
    fi
done

if [[ "${IS_AUTHORIZED}" != "true" ]]; then
    echo "Sender ${SENDER} is not a member of any authorized team"
    exit 0
fi

HEAD_SHA=$(gh pr view "${PR_NUMBER}" --json headRefOid --jq '.headRefOid')

echo "Finding workflows pending approval for commit: ${HEAD_SHA}"

WAITING_RUNS=$(gh run list \
    --commit "${HEAD_SHA}" \
    --json databaseId,status,conclusion \
    --jq '.[] | select(.conclusion == "action_required") | .databaseId')

if [[ -z "${WAITING_RUNS}" ]]; then
    echo "No workflows with action_required conclusion found"
    exit 0
fi

for RUN_ID in ${WAITING_RUNS}; do
    echo "Approving workflow run: ${RUN_ID}"
    gh api --method POST "repos/${GITHUB_REPOSITORY}/actions/runs/${RUN_ID}/approve" --silent
done
