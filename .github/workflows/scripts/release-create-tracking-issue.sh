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

EXISTING_ISSUE=$( gh issue list --search "Release v${CANDIDATE_BETA}/v${CANDIDATE_STABLE}" --json url --jq '.[].url' --repo "${REPO}" )

if [ "${EXISTING_ISSUE}" != "" ]; then
    echo "Issue already exists: ${EXISTING_ISSUE}"
    exit 0
fi

gh issue create -a "${GITHUB_ACTOR}" --repo "${REPO}" --label release --title "Release v${CANDIDATE_BETA}/v${CANDIDATE_STABLE}" --body "Like #4522, but for v${CANDIDATE_BETA}/v${CANDIDATE_STABLE}
**Performed by collector release manager**

- [ ] Prepare stable core release v${CANDIDATE_STABLE}
- [ ] Prepare beta core release v${CANDIDATE_BETA}
- [ ] Tag and release stable core v${CANDIDATE_STABLE}
- [ ] Tag and release beta core v${CANDIDATE_BETA}
- [ ] Prepare contrib release v${CANDIDATE_BETA}
- [ ] Tag and release contrib v${CANDIDATE_BETA}
- [ ] Prepare otelcol-releases v${CANDIDATE_BETA}
- [ ] Release binaries and container images v${CANDIDATE_BETA}

**Performed by operator maintainers**

- [ ] Release the operator v${CANDIDATE_BETA}

**Performed by helm chart maintainers**

- [ ] Update the opentelemetry-collector helm chart to use v${CANDIDATE_BETA}"
