#!/bin/bash -ex

BLOCKERS=$( gh issue list --search "label:release:blocker" --json url --jq '.[].url' --repo "${REPO}" )
if [ "${BLOCKERS}" != "" ]; then
    echo "Release blockers in ${REPO} repo: ${BLOCKERS}"
    exit 1
fi
