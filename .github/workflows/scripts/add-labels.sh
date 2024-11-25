#!/usr/bin/env bash
#
# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0
#
#

set -euo pipefail

if [[ -z "${ISSUE:-}" || -z "${COMMENT:-}" || -z "${SENDER:-}" ]]; then
    echo "At least one of ISSUE, COMMENT, or SENDER has not been set, please ensure each is set."
    exit 0
fi

CUR_DIRECTORY=$(dirname "$0")

if [[ ${COMMENT:0:6} != "/label" ]]; then
    echo "Comment is not a label comment, exiting."
    exit 0
fi

declare -A COMMON_LABELS
COMMON_LABELS["arm64"]="arm64"
COMMON_LABELS["good-first-issue"]="good first issue"
COMMON_LABELS["help-wanted"]="help wanted"
COMMON_LABELS["needs-discussion"]="needs discussion"
COMMON_LABELS["needs-triage"]="needs triage"
COMMON_LABELS["os:mac"]="os:mac"
COMMON_LABELS["os:windows"]="os:windows"
COMMON_LABELS["waiting-for-author"]="waiting for author"

LABELS=$(echo "${COMMENT}" | sed -E 's%^/label%%')

for LABEL_REQ in ${LABELS}; do
    LABEL=$(echo "${LABEL_REQ}" | sed -E s/^[+-]?//)
    # Trim newlines from label that would cause matching to fail
    LABEL=$(echo "${LABEL}" | tr -d '\n')

    SHOULD_ADD=true
    if [[ "${LABEL_REQ:0:1}" = "-" ]]; then
        SHOULD_ADD=false
    fi

    if [[ -v COMMON_LABELS["${LABEL}"] ]]; then
        if [[ ${SHOULD_ADD} = true ]]; then
            gh issue edit "${ISSUE}" --add-label "${COMMON_LABELS["${LABEL}"]}"
        else
            gh issue edit "${ISSUE}" --remove-label "${COMMON_LABELS["${LABEL}"]}"
        fi
        continue
    fi

    # Grep exits with status code 1 if there are no matches,
    # so we manually set RESULT to 0 if nothing is found.
    RESULT=$(grep -c "${LABEL}" .github/CODEOWNERS || true)

    if [[ ${RESULT} = 0 ]]; then
        echo "\"${LABEL}\" doesn't correspond to a component, skipping."
        continue
    fi

    if [[ ${SHOULD_ADD} = true ]]; then
        gh issue edit "${ISSUE}" --add-label "${LABEL}"

        # Labels added by a GitHub Actions workflow don't trigger other workflows
        # by design, so we have to manually ping code owners here.
        COMPONENT="${LABEL}" ISSUE=${ISSUE} SENDER="${SENDER}" bash "${CUR_DIRECTORY}/ping-codeowners-issues.sh"
    else
        gh issue edit "${ISSUE}" --remove-label "${LABEL}"
    fi
done


