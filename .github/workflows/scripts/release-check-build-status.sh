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

BRANCH=main
WORKFLOW=build-and-test

RESULT=$(gh run list --branch "${BRANCH}" --json status --jq '[.[] | select(.status != "queued" and .status != "in_progress")][0].status' --workflow "${WORKFLOW}" --repo "${REPO}" )
if [ "${RESULT}" != "completed" ]; then
    echo "Build status in ${REPO} is not completed: ${RESULT}"
    gh run list --branch "${BRANCH}" --json status,url --jq '[.[] | select(.status != "queued" and .status != "in_progress")][0].url' --workflow "${WORKFLOW}" --repo "${REPO}"
    exit 1
fi
