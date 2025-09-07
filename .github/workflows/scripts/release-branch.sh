#!/bin/bash -ex
#
# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0

# --- Configuration ---
UPSTREAM_REMOTE_NAME=${UPSTREAM_REMOTE_NAME:-"upstream"} # Your upstream remote name for open-telemetry/opentelemetry-collector
MAIN_BRANCH_NAME=${MAIN_BRANCH_NAME:-"main"}
LOCAL_MAIN_BRANCH_NAME=${LOCAL_MAIN_BRANCH_NAME:-"${MAIN_BRANCH_NAME}"}
# These variables are only used if git user.name and git user.email are not configured
GIT_CONFIG_USER_NAME=${GIT_CONFIG_USER_NAME:-"otelbot"}
GIT_CONFIG_USER_EMAIL=${GIT_CONFIG_USER_EMAIL:-"197425009+otelbot@users.noreply.github.com"}

# --- Extract release information from tag ---
if [[ -z "$GITHUB_REF" ]]; then
  echo "Error: GITHUB_REF environment variable must be provided when running in GitHub Actions."
  echo "For manual usage: GITHUB_REF=refs/tags/v0.85.0 $0"
  exit 1
fi

# Extract tag name and validate format using regex
if [[ ! $GITHUB_REF =~ ^refs/tags/v([0-9]+\.[0-9]+)\.[0-9]+(-.+)?$ ]]; then
    echo "Error: GITHUB_REF did not match expected format (refs/tags/vX.XX.X)"
    exit 1
fi

# Extract version numbers from regex match
VERSION_MAJOR_MINOR=${BASH_REMATCH[1]}
RELEASE_SERIES="v${VERSION_MAJOR_MINOR}.x"
echo "Release series: ${RELEASE_SERIES}"

# --- Use current commit as prepare release commit ---
PREPARE_RELEASE_COMMIT_HASH="${GITHUB_SHA:-HEAD}"
echo "Using current commit as prepare release commit: ${PREPARE_RELEASE_COMMIT_HASH}"

RELEASE_BRANCH_NAME="release/${RELEASE_SERIES}"

echo "Automating Release Steps for: ${RELEASE_SERIES}"
echo "Release Branch Name: ${RELEASE_BRANCH_NAME}"
echo "'Prepare release' commit hash: ${PREPARE_RELEASE_COMMIT_HASH}"
echo "Upstream Remote: ${UPSTREAM_REMOTE_NAME}"
echo "--------------------------------------------------"

# --- Step 4: Checkout main, Pull, Create/Update and Push Release Branch ---
echo ""
echo "=== Step 4: Preparing and Pushing Release Branch ==="

# 1. Checkout main
git checkout "${LOCAL_MAIN_BRANCH_NAME}"

# 2. Fetch from upstream (updates remote-tracking branches including potential existing release branch)
git fetch "${UPSTREAM_REMOTE_NAME}"

# 3. Rebase local main with upstream/main
git rebase "${UPSTREAM_REMOTE_NAME}/${MAIN_BRANCH_NAME}"
echo "'${LOCAL_MAIN_BRANCH_NAME}' branch is now up-to-date."

# Verify the commit exists (it should be reachable from main after fetch)
if ! git cat-file -e "${PREPARE_RELEASE_COMMIT_HASH}^{commit}" 2>/dev/null; then
  echo "Error: Provided 'Prepare release' commit hash '${PREPARE_RELEASE_COMMIT_HASH}' not found."
  exit 1
fi
# 4. Handle Release Branch: Check existence, create or switch, and merge/base
BRANCH_EXISTS_LOCALLY=$(git branch --list "${RELEASE_BRANCH_NAME}")
# Check remote by looking for the remote tracking branch ref after fetch
if git rev-parse --verify --quiet "${UPSTREAM_REMOTE_NAME}/${RELEASE_BRANCH_NAME}" > /dev/null 2>&1; then
  BRANCH_EXISTS_REMOTELY=true
fi

if [[ -n "$BRANCH_EXISTS_LOCALLY" ]]; then
  echo "Release branch '${RELEASE_BRANCH_NAME}' found locally."
  echo "Please delete local release branch using 'git branch -D ${RELEASE_BRANCH_NAME}' and run the script again."
  exit 1
elif [[ -n "$BRANCH_EXISTS_REMOTELY" ]]; then
  echo "Release branch '${RELEASE_BRANCH_NAME}' found on remote '${UPSTREAM_REMOTE_NAME}'."
  echo "Nothing to do, exiting."
  exit 0
else
  echo "Release branch '${RELEASE_BRANCH_NAME}' not found locally or on remote."
  git switch -c "${RELEASE_BRANCH_NAME}" "${PREPARE_RELEASE_COMMIT_HASH}"
fi

echo "Current branch is now '${RELEASE_BRANCH_NAME}'."
git --no-pager log -1 --pretty=oneline # Show the commit at the tip of the release branch

# 5. Push the release branch to upstream
git push -u "${UPSTREAM_REMOTE_NAME}" "${RELEASE_BRANCH_NAME}"
echo "Branch '${RELEASE_BRANCH_NAME}' pushed (or updated) on '${UPSTREAM_REMOTE_NAME}'."
echo "Step 4 completed."
echo "--------------------------------------------------"

echo ""
echo "Automation for release branch creation complete."
echo "Release branch '${RELEASE_BRANCH_NAME}' has been created from the prepare release commit."
echo "Tag-triggered build workflows should now be running for the pushed tags."
