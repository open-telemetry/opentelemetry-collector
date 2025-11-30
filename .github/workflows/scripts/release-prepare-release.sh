#!/bin/bash -ex
#
# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0

if [ "${CANDIDATE_STABLE}" == "" ] && [ "${CANDIDATE_BETA}" == "" ]; then
    echo "One of CANDIDATE_STABLE or CANDIDATE_BETA must be set"
    exit 1
fi

# Expand CURRENT_STABLE and CURRENT_BETA to escape . character by using [.]
CURRENT_STABLE_ESCAPED=${CURRENT_STABLE//./[.]}
CURRENT_BETA_ESCAPED=${CURRENT_BETA//./[.]}

RELEASE_VERSION=v${CANDIDATE_STABLE}/v${CANDIDATE_BETA}
if [ "${CANDIDATE_STABLE}" == "" ]; then
    RELEASE_VERSION="v${CANDIDATE_BETA}"
fi
if [ "${CANDIDATE_BETA}" == "" ]; then
    RELEASE_VERSION="v${CANDIDATE_STABLE}"
fi

make chlog-update VERSION="${RELEASE_VERSION}"
COMMANDS="- make chlog-update VERSION=${RELEASE_VERSION}"
git config user.name otelbot
git config user.email 197425009+otelbot@users.noreply.github.com
BRANCH="prepare-release-prs/${CANDIDATE_BETA}"
git checkout -b "${BRANCH}"
git add --all
git commit -m "Changelog update ${RELEASE_VERSION}"

if [ "${CANDIDATE_STABLE}" != "" ]; then
    make prepare-release PREVIOUS_VERSION="${CURRENT_STABLE_ESCAPED}" RELEASE_CANDIDATE="${CANDIDATE_STABLE}" MODSET=stable
    COMMANDS+="
- make prepare-release PREVIOUS_VERSION=${CURRENT_STABLE_ESCAPED} RELEASE_CANDIDATE=${CANDIDATE_STABLE} MODSET=stable"
fi
if [ "${CANDIDATE_BETA}" != "" ]; then
    make prepare-release PREVIOUS_VERSION="${CURRENT_BETA_ESCAPED}" RELEASE_CANDIDATE="${CANDIDATE_BETA}" MODSET=beta
    COMMANDS+="
- make prepare-release PREVIOUS_VERSION=${CURRENT_BETA_ESCAPED} RELEASE_CANDIDATE=${CANDIDATE_BETA} MODSET=beta"
fi
git push --set-upstream origin "${BRANCH}"

# Use OpenTelemetryBot account to create PR, allowing workflows to run
# The title must match the checks in check-merge-freeze.yml
gh pr create --head "$(git branch --show-current)" --title "[chore] Prepare release ${RELEASE_VERSION}" --body "
The following commands were run to prepare this release:
${COMMANDS}
"
