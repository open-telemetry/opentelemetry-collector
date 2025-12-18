#!/usr/bin/env bash
#
# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0
#

set -euo pipefail

if [[ -z "${PR_NUMBER:-}" || -z "${COMMENT:-}" ]]; then
    echo "PR_NUMBER or COMMENT not set"
    exit 0
fi

if [[ ${COMMENT:0:6} != "/rerun" ]]; then
    echo "Not a rerun command"
    exit 0
fi

HEAD_SHA=$(gh pr view "${PR_NUMBER}" --json headRefOid --jq .headRefOid)

echo "Finding failed workflows for commit: ${HEAD_SHA}"

FAILED_RUNS=$(gh run list \
    --commit "${HEAD_SHA}" \
    --status failure \
    --json databaseId \
    --jq '.[].databaseId')

if [[ -z "${FAILED_RUNS}" ]]; then
    echo "No failed workflows found"
    exit 0
else
    for RUN_ID in ${FAILED_RUNS}; do
        echo "Rerunning workflow: ${RUN_ID}"
        gh run rerun "${RUN_ID}" --failed
    done
fi
