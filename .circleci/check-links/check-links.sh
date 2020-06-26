#!/bin/bash

set -eu

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
REPO_DIR="$( cd "${SCRIPT_DIR}/../../" && pwd )"
CIRCLE_BRANCH=${CIRCLE_BRANCH:-}
CIRCLE_TAG=${CIRCLE_TAG:-}

nfailed=0

# check all docs in master/tags or new/modified docs in PR
for md in $(find "$REPO_DIR" -name "*.md" | sort); do
    if [[ "$CIRCLE_BRANCH" = "master" ]] || [[ -n "$CIRCLE_TAG" ]] || [[ -n "$( git -c "$REPO_DIR" diff HEAD origin/master --name-only | grep "^${md/#$REPO_DIR\//}" )" ]]; then
        markdown-link-check -c ${SCRIPT_DIR}/config.json -qv "$md" || (( nfailed += $? ))
        # wait to scan files so that we don't overload github with requests which may result in 429 responses
        sleep 2
    fi
done

exit $nfailed
