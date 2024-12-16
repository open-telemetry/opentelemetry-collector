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
git config user.name opentelemetrybot
git config user.email 107717825+opentelemetrybot@users.noreply.github.com
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
git push origin "${BRANCH}"

# Use OpenTelemetryBot account to create PR, allowing workflows to run
PR=$(GITHUB_TOKEN="$BOT_GITHUB_TOKEN" gh pr create --title "[chore] Prepare release ${RELEASE_VERSION}" --body "
The following commands were run to prepare this release:
${COMMANDS}
")

# The `release:merge-freeze` label will cause the `check-merge-freeze` workflow to fail, enforcing the freeze.
# The bot does not have permissions to add labels, so this is done using the CI action token.
gh pr edit "$PR" --add-label release:merge-freeze || echo "Failed to add merge-freeze label"
