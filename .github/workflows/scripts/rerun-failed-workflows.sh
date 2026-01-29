#!/usr/bin/env bash
#
# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0
#

set -euo pipefail

if [[ -z "${PR_NUMBER:-}" || -z "${COMMENT:-}" || -z "${SENDER:-}" ]]; then
    echo "PR_NUMBER, COMMENT, or SENDER not set"
    exit 0
fi

if [[ ${COMMENT:0:6} != "/rerun" ]]; then
    echo "Not a rerun command"
    exit 0
fi

PR_DATA=$(gh pr view "${PR_NUMBER}" --json headRefOid,author)
HEAD_SHA=$(echo "${PR_DATA}" | jq -r '.headRefOid')
PR_AUTHOR=$(echo "${PR_DATA}" | jq -r '.author.login')

if [[ "${SENDER}" != "${PR_AUTHOR}" ]]; then
    echo "Only PR author can rerun workflows"
    exit 0
fi

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
