#!/bin/bash -ex
#
# Copyright The OpenTelemetry Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

if [ "${CANDIDATE_STABLE}" == "" ] && [ "${CANDIDATE_BETA}" == "" ]; then
    echo "One of CANDIDATE_STABLE or CANDIDATE_BETA must be set"
    exit 1
fi

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
    make prepare-release GH=none PREVIOUS_VERSION="${CURRENT_STABLE}" RELEASE_CANDIDATE="${CANDIDATE_STABLE}" MODSET=stable
    COMMANDS+="
- make prepare-release GH=none PREVIOUS_VERSION=${CURRENT_STABLE} RELEASE_CANDIDATE=${CANDIDATE_STABLE} MODSET=stable"
fi
if [ "${CANDIDATE_BETA}" != "" ]; then
    make prepare-release GH=none PREVIOUS_VERSION="${CURRENT_BETA}" RELEASE_CANDIDATE="${CANDIDATE_BETA}" MODSET=beta
    COMMANDS+="
- make prepare-release GH=none PREVIOUS_VERSION=${CURRENT_BETA} RELEASE_CANDIDATE=${CANDIDATE_BETA} MODSET=beta"
fi
git push origin "${BRANCH}"

gh pr create --title "[chore] Prepare release ${RELEASE_VERSION}" --body "
The following commands were run to prepare this release:
${COMMANDS}
"
