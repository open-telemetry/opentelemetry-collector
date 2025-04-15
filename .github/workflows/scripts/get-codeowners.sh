#!/usr/bin/env bash
#
# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0
#
# This script checks the GitHub CODEOWNERS file for any code owners
# of contrib components and returns a string of the code owners if it
# finds them.

set -euo pipefail

get_component_type() {
  echo "${COMPONENT}" | cut -f 1 -d '/'
}

get_codeowners() {
  # grep arguments explained:
  #   -m 1: Match the first occurrence
  #   ^: Match from the beginning of the line
  #   ${1}: Insert first argument given to this function
  #   [\/]\?: Match 0 or 1 instances of a forward slash
  #   \s: Match any whitespace character
(grep -m 1 "^${1}[\/]\?\s" .github/CODEOWNERS || true) | \
        sed 's/   */ /g' | \
        cut -f3- -d ' '
}

if [[ -z "${COMPONENT:-}" ]]; then
    echo "COMPONENT has not been set, please ensure it is set."
    exit 1
fi

OWNERS="$(get_codeowners "${COMPONENT}")"

if [[ -z "${OWNERS:-}" ]]; then
    COMPONENT_TYPE=$(get_component_type "${COMPONENT}")
    OWNERS="$(get_codeowners "${COMPONENT}${COMPONENT_TYPE}")"
fi

echo "${OWNERS}"
