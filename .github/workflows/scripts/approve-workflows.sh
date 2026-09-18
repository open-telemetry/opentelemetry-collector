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

# Runs do not become queryable as action_required the instant they are created,
# so a comment posted right after a push can find nothing and give up. Poll with
# the same backoff the approval loop below uses.
DELAY=5
WAITING_RUNS=""
while :; do
    WAITING_RUNS=$(gh run list \
        --commit "${HEAD_SHA}" \
        --status action_required \
        --limit 100 \
        --json databaseId \
        --jq '.[].databaseId')

    [[ -n "${WAITING_RUNS}" ]] && break
    [[ ${DELAY} -gt 60 ]] && break

    echo "No runs pending approval yet for ${HEAD_SHA}; retrying in ${DELAY}s..."
    sleep "${DELAY}"
    DELAY=$(( DELAY * 2 ))
done

if [[ -z "${WAITING_RUNS}" ]]; then
    echo "No workflows pending approval for ${HEAD_SHA} after waiting"
    exit 0
fi

for RUN_ID in ${WAITING_RUNS}; do
    echo "Approving workflow run: ${RUN_ID}"
    DELAY=5
    APPROVED=false
    while [[ ${DELAY} -le 60 ]]; do
        if gh api --method POST "repos/${GITHUB_REPOSITORY}/actions/runs/${RUN_ID}/approve" --silent; then
            APPROVED=true
            break
        fi
        echo "Approval failed for run ${RUN_ID}, retrying in ${DELAY}s..."
        sleep "${DELAY}"
        DELAY=$(( DELAY * 2 ))
    done
    if [[ "${APPROVED}" != "true" ]]; then
        echo "Failed to approve run ${RUN_ID} after retries"
        exit 1
    fi
done
