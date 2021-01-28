#!/bin/bash

# Copyright The OpenTelemetry Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#       http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -eu

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
REPO_DIR="$( cd "${SCRIPT_DIR}/../../../" && pwd )"
GITHUB_REF=${GITHUB_REF:-}

diff_files="$( git diff HEAD origin/main --name-only )"
check_all_files=1
if [[ "$GITHUB_REF" = "ref/heads/main" ]] || [[ -n "$( echo "$diff_files" | grep ".github/workflows/check-links" )" ]]; then
    check_all_files=0
fi

nfailed=0

# check all docs in main/tags or new/modified docs in PR
for md in $(find "$REPO_DIR" -name "*.md" | sort); do
    if [[ $check_all_files ]] || [[ -n "$( echo "$diff_files" | grep "^${md/#$REPO_DIR\//}" )" ]]; then
        node $SCRIPT_DIR/markdown-link-check -c ${SCRIPT_DIR}/config.json -v "$md" || (( nfailed += $? ))
        # wait to scan files so that we don't overload github with requests which may result in 429 responses
        sleep 2
    fi
done

exit $nfailed
