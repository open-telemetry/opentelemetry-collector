#!/bin/bash -e

BLOCKERS=$(gh pr list --search "label:release:prepare" --json url --jq '.[].url' --repo "${REPO}")
if [ "${BLOCKERS}" != "" ]; then
  echo "Merging in main is frozen by release PR: ${BLOCKERS}"
  exit 1
fi
