#!/usr/bin/env bash
#
# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0
#
#

set -euo pipefail

if [[ -z "${ISSUE:-}" || -z "${TITLE:-}" || -z "${BODY:-}" || -z "${OPENER:-}" ]]; then
  echo "Missing one of ISSUE, TITLE, BODY, or OPENER, please ensure all are set."
  exit 0
fi

LABELS_COMMENT='See [Adding Labels via Comments](https://github.com/open-telemetry/opentelemetry-collector/blob/main/CONTRIBUTING.md#adding-labels-via-comments) if you do not have permissions to add labels yourself.'
CUR_DIRECTORY=$(dirname "$0")
LABELS=""
PING_LINES=""
declare -A PINGED_COMPONENTS

TITLE_COMPONENT=$(echo "${TITLE}" | (grep -oE "\[.+\]" || true) | sed -E 's/\[(.+)\]/\1/' | sed -E 's%^(.+)/(.+)\1%\1/\2%')

COMPONENTS_SECTION_START=$( (echo "${BODY}" | grep -n '### Component(s)' | awk '{ print $1 }' | grep -oE '[0-9]+') || echo '-1' )
BODY_COMPONENTS=""

if [[ "${COMPONENTS_SECTION_START}" != '-1' ]]; then
  BODY_COMPONENTS=$(echo "${BODY}" | sed -n $((COMPONENTS_SECTION_START+2))p)
fi

if [[ -n "${TITLE_COMPONENT}" && ! ("${TITLE_COMPONENT}" =~ " ") ]]; then
  CODEOWNERS=$(COMPONENT="${TITLE_COMPONENT}" "${CUR_DIRECTORY}/get-codeowners.sh" || true)

  if [[ -n "${CODEOWNERS}" ]]; then
    PING_LINES+="- ${TITLE_COMPONENT}: ${CODEOWNERS}\n"
    PINGED_COMPONENTS["${TITLE_COMPONENT}"]=1

    if (( "${#TITLE_COMPONENT}" <= 50 )); then
      LABELS+="${TITLE_COMPONENT}"
    else
      echo "'${TITLE_COMPONENT}' exceeds GitHub's 50-character limit, skipping adding a label"
    fi
  fi
fi

for COMPONENT in ${BODY_COMPONENTS}; do
  # Comments are delimited by ', ' and the for loop separates on spaces, so remove the extra comma.
  COMPONENT=${COMPONENT//,/}

  CODEOWNERS=$(COMPONENT="${COMPONENT}" "${CUR_DIRECTORY}/get-codeowners.sh" || true)

  if [[ -n "${CODEOWNERS}" ]]; then
    if [[ -v PINGED_COMPONENTS["${COMPONENT}"] ]]; then
      continue
    fi

    PING_LINES+="- ${COMPONENT}: ${CODEOWNERS}\n"
    PINGED_COMPONENTS["${COMPONENT}"]=1

    if (( "${#COMPONENT}" > 50 )); then
      echo "'${COMPONENT}' exceeds GitHub's 50-character limit on labels, skipping adding a label"
      continue
    fi

    if [[ -n "${LABELS}" ]]; then
      LABELS+=","
    fi
    LABELS+="${COMPONENT}"
  fi
done

if [[ -v PINGED_COMPONENTS[@] ]]; then
  echo "The issue was associated with components:" "${!PINGED_COMPONENTS[@]}"
else
  echo "No related components were given"
fi

if [[ -n "${LABELS}" ]]; then
  # Notes on this call:
  # 1. Labels will be deduplicated by the GitHub CLI.
  # 2. The call to edit the issue will fail if any of the
  #    labels doesn't exist. We can be reasonably sure that
  #    all labels will exist since they come from a known set.
  echo "Adding the following labels: ${LABELS//,/ /}"
  gh issue edit "${ISSUE}" --add-label "${LABELS}" || true
else
  echo "No labels were found to add"
fi

if [[ -n "${PING_LINES}" ]]; then
  # Notes on this call:
  # 1. Adding labels above will not trigger the ping-codeowners flow,
  #    since GitHub Actions disallows triggering a workflow from a
  #    workflow, so we have to ping code owners here.
  # 2. The GitHub CLI only offers multiline strings through file input,
  #    so we provide the comment through stdin.
  # 3. The PING_LINES variable must be directly put into the echo string
  #    to get the newlines to render correctly, using string formatting
  #    causes the newlines to be interpreted literally.
  echo -e "Pinging code owners:\n${PING_LINES}"
  echo -e "Pinging code owners:\n${PING_LINES}\n" "${LABELS_COMMENT}"  \
  | gh issue comment "${ISSUE}" -F -
else
  echo "No code owners were found to ping"
fi
