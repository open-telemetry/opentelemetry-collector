#!/bin/bash -ex

BRANCH=main
WORKFLOW=build-and-test

RESULT=$(gh run list --branch "${BRANCH}" --json status --jq '[.[] | select(.status != "queued" and .status != "in_progress")][0].status' --workflow "${WORKFLOW}" --repo "${REPO}" )
if [ "${RESULT}" != "completed" ]; then
    echo "Build status in ${REPO} is not completed: ${RESULT}"
    gh run list --branch "${BRANCH}" --json status,url --jq '[.[] | select(.status != "queued" and .status != "in_progress")][0].url' --workflow "${WORKFLOW}" --repo "${REPO}"
    exit 1
fi
