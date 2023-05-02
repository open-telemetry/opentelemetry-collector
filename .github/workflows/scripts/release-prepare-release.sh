#!/bin/bash -ex
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

make chlog-update VERSION="v${CANDIDATE_STABLE}/v${CANDIDATE_BETA}"
git config user.name opentelemetrybot
git config user.email 107717825+opentelemetrybot@users.noreply.github.com
BRANCH="prepare-release-prs/${CANDIDATE_BETA}-${CANDIDATE_STABLE}"
git checkout -b "${BRANCH}"
git add --all
git commit -m "Changelog update ${CANDIDATE_BETA}/${CANDIDATE_STABLE}"

make prepare-release GH=none PREVIOUS_VERSION="${CURRENT_STABLE}" RELEASE_CANDIDATE="${CANDIDATE_STABLE}" MODSET=stable
make prepare-release GH=none PREVIOUS_VERSION="${CURRENT_BETA}" RELEASE_CANDIDATE="${CANDIDATE_BETA}" MODSET=beta
git push origin "${BRANCH}"

gh pr create --title "[chore] Prepare release ${CANDIDATE_BETA}/${CANDIDATE_STABLE}" --body "
The following commands were run to prepare this release:
- make chlog-update VERSION=v${CANDIDATE_BETA}/v${CANDIDATE_STABLE}
- make prepare-release GH=none PREVIOUS_VERSION=${CURRENT_STABLE} RELEASE_CANDIDATE=${CANDIDATE_STABLE} MODSET=stable
- make prepare-release GH=none PREVIOUS_VERSION=${CURRENT_BETA} RELEASE_CANDIDATE=${CANDIDATE_BETA} MODSET=beta
"
