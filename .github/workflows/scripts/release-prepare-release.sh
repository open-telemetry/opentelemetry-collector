#!/bin/bash

make chlog-update VERSION="${CANDIDATE_STABLE}/${CANDIDATE_BETA}"
git config user.name "$GITHUB_ACTOR"
git config user.email "$GITHUB_ACTOR@users.noreply.github.com"
BRANCH="prepare-release-prs/${CANDIDATE_BETA}-${CANDIDATE_STABLE}"
git checkout -b "${BRANCH}"
git add --all
git commit -m "Changelog update ${CANDIDATE_BETA}/${CANDIDATE_STABLE}"

make prepare-release GH=none PREVIOUS_VERSION="${CURRENT_STABLE}" RELEASE_CANDIDATE="${CANDIDATE_STABLE}" MODSET=stable
make prepare-release GH=none PREVIOUS_VERSION="${CURRENT_BETA}" RELEASE_CANDIDATE="${CANDIDATE_BETA}" MODSET=beta
git push origin "${BRANCH}"

gh pr create --title "[chore] Prepare release ${CANDIDATE_BETA}/${CANDIDATE_STABLE}" --body "
The following commands were run to prepare this release:
- make chlog-update VERSION=${CANDIDATE_BETA}/${CANDIDATE_STABLE}
- make prepare-release GH=none PREVIOUS_VERSION=${CURRENT_STABLE} RELEASE_CANDIDATE=${CANDIDATE_STABLE} MODSET=stable
- make prepare-release GH=none PREVIOUS_VERSION=${CURRENT_BETA} RELEASE_CANDIDATE=${CANDIDATE_BETA} MODSET=beta
"
