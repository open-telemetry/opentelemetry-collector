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

RELEASE_VERSION=v${CANDIDATE_STABLE}/v${CANDIDATE_BETA}
if [ "${CANDIDATE_STABLE}" == "" ]; then
    RELEASE_VERSION="v${CANDIDATE_BETA}"
fi
if [ "${CANDIDATE_BETA}" == "" ]; then
    RELEASE_VERSION="v${CANDIDATE_STABLE}"
fi

EXISTING_ISSUE=$( gh issue list --search "Release ${RELEASE_VERSION}" --json url --jq '.[].url' --repo "${REPO}" )

if [ "${EXISTING_ISSUE}" != "" ]; then
    echo "Issue already exists: ${EXISTING_ISSUE}"
    exit 0
fi

gh issue create -a "${GITHUB_ACTOR}" --repo "${REPO}" --label release --title "Release ${RELEASE_VERSION}" --body "Like #4522, but for ${RELEASE_VERSION}
**Performed by collector release manager**

- [ ] Prepare core release ${RELEASE_VERSION}
- [ ] Tag and release core ${RELEASE_VERSION}
- [ ] Prepare contrib release v${CANDIDATE_BETA}
- [ ] Tag and release contrib v${CANDIDATE_BETA}
- [ ] Prepare otelcol-releases v${CANDIDATE_BETA}
- [ ] Release binaries and container images v${CANDIDATE_BETA}

**Performed by operator maintainers**

- [ ] Release the operator v${CANDIDATE_BETA}

**Performed by helm chart maintainers**

- [ ] Update the opentelemetry-collector helm chart to use v${CANDIDATE_BETA}"
