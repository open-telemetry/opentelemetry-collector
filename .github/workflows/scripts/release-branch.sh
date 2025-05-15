#!/bin/bash -ex
#
# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0

# --- Configuration ---
RELEASE_SERIES="$1" # e.g., v0.85.x
PREPARE_RELEASE_COMMIT_HASH="$2" # Optional: Specific commit hash for "Prepare release"
UPSTREAM_REMOTE_NAME=${UPSTREAM_REMOTE_NAME:-"upstream"} # Your upstream remote name for open-telemetry/opentelemetry-collector
MAIN_BRANCH_NAME=${MAIN_BRANCH_NAME:-"main"}
LOCAL_MAIN_BRANCH_NAME=${LOCAL_MAIN_BRANCH_NAME:-"${MAIN_BRANCH_NAME}"}
GIT_CONFIG_USER_NAME=${GIT_CONFIG_USER_NAME:-"opentelemetrybot"}
GIT_CONFIG_USER_EMAIL=${GIT_CONFIG_USER_EMAIL:-"107717825+opentelemetrybot@users.noreply.github.com"}

# --- Helper Functions ---
check_command_success() {
  if [[ $? -ne 0 ]]; then
    echo "Error: Command failed - '$1'. See output above for details."
    exit 1
  fi
}

# --- Validate Input ---
if [[ -z "$RELEASE_SERIES" ]]; then
  echo "Error: Release series not provided."
  echo "Usage: $0 <release-series> [prepare-release-commit-hash]"
  echo "Example: $0 v0.85.x"
  echo "Example: $0 v0.85.x a1b2c3d4"
  exit 1
fi

RELEASE_BRANCH_NAME="release/${RELEASE_SERIES}"

echo "Automating Release Steps for: ${RELEASE_SERIES}"
echo "Release Branch Name: ${RELEASE_BRANCH_NAME}"
if [[ -n "$PREPARE_RELEASE_COMMIT_HASH" ]]; then
  echo "Will use specific 'Prepare release' commit: ${PREPARE_RELEASE_COMMIT_HASH}"
else
  echo "Will use HEAD of '${UPSTREAM_REMOTE_NAME}/${MAIN_BRANCH_NAME}' (after fetch/rebase) as 'Prepare release' commit."
fi
echo "Upstream Remote: ${UPSTREAM_REMOTE_NAME}"
echo "--------------------------------------------------"

# --- Step 4: Checkout main, Pull, Create/Update and Push Release Branch ---
echo ""
echo "=== Step 4: Preparing and Pushing Release Branch ==="

# 1. Checkout main
echo "1. Checking out '${MAIN_BRANCH_NAME}' branch..."
git config user.name "${GIT_CONFIG_USER_NAME}"
git config user.email "${GIT_CONFIG_USER_EMAIL}"
git checkout "${LOCAL_MAIN_BRANCH_NAME}"
check_command_success "git checkout ${LOCAL_MAIN_BRANCH_NAME}"

# 2. Fetch from upstream (updates remote-tracking branches including potential existing release branch)
echo "2. Fetching from '${UPSTREAM_REMOTE_NAME}'..."
git fetch "${UPSTREAM_REMOTE_NAME}"
check_command_success "git fetch ${UPSTREAM_REMOTE_NAME}"

# 3. Rebase local main with upstream/main
echo "3. Rebasing local '${MAIN_BRANCH_NAME}' with '${UPSTREAM_REMOTE_NAME}/${MAIN_BRANCH_NAME}'..."
git rebase "${UPSTREAM_REMOTE_NAME}/${MAIN_BRANCH_NAME}"
check_command_success "git rebase ${UPSTREAM_REMOTE_NAME}/${MAIN_BRANCH_NAME}"
echo "'${LOCAL_MAIN_BRANCH_NAME}' branch is now up-to-date."

# Determine the target commit from main that the release branch should be based on or merge
TARGET_MAIN_STATE_COMMIT=""
if [[ -n "$PREPARE_RELEASE_COMMIT_HASH" ]]; then
  TARGET_MAIN_STATE_COMMIT="$PREPARE_RELEASE_COMMIT_HASH"
  # Verify the commit exists (it should be reachable from main after fetch)
  if ! git cat-file -e "${TARGET_MAIN_STATE_COMMIT}^{commit}" 2>/dev/null; then
    echo "Error: Provided 'Prepare release' commit hash '${TARGET_MAIN_STATE_COMMIT}' not found."
    exit 1
  fi
  echo "Targeting specific commit from main: ${TARGET_MAIN_STATE_COMMIT}"
else
  # If no specific commit is given, use the current HEAD of the rebased local main branch
  TARGET_MAIN_STATE_COMMIT=$(git rev-parse HEAD) # Assumes we are still on 'main'
  echo "Targeting current HEAD of rebased '${MAIN_BRANCH_NAME}': ${TARGET_MAIN_STATE_COMMIT}"
fi

# 4. Handle Release Branch: Check existence, create or switch, and merge/base
echo "4. Checking for existing release branch '${RELEASE_BRANCH_NAME}'..."
BRANCH_EXISTS_LOCALLY=$(git branch --list "${RELEASE_BRANCH_NAME}")
# Check remote by looking for the remote tracking branch ref after fetch
if git rev-parse --verify --quiet "${UPSTREAM_REMOTE_NAME}/${RELEASE_BRANCH_NAME}" > /dev/null 2>&1; then
  BRANCH_EXISTS_REMOTELY=true
else
  BRANCH_EXISTS_REMOTELY=
fi

MERGE_NEEDED=false

if [[ -n "$BRANCH_EXISTS_REMOTELY" ]]; then
  echo "Release branch '${RELEASE_BRANCH_NAME}' found on remote '${UPSTREAM_REMOTE_NAME}'."
  MERGE_NEEDED=true
  if [[ -n "$BRANCH_EXISTS_LOCALLY" ]]; then
    echo "Switching to existing local branch '${RELEASE_BRANCH_NAME}'."
    git switch "${RELEASE_BRANCH_NAME}"
    check_command_success "git switch ${RELEASE_BRANCH_NAME}"
    echo "Resetting local '${RELEASE_BRANCH_NAME}' to match '${UPSTREAM_REMOTE_NAME}/${RELEASE_BRANCH_NAME}' to ensure consistency."
    git reset --hard "${UPSTREAM_REMOTE_NAME}/${RELEASE_BRANCH_NAME}"
    check_command_success "git reset --hard ${UPSTREAM_REMOTE_NAME}/${RELEASE_BRANCH_NAME}"
  else
    echo "Creating local branch '${RELEASE_BRANCH_NAME}' to track '${UPSTREAM_REMOTE_NAME}/${RELEASE_BRANCH_NAME}'."
    git switch -c "${RELEASE_BRANCH_NAME}" "${UPSTREAM_REMOTE_NAME}/${RELEASE_BRANCH_NAME}"
    check_command_success "git switch -c ${RELEASE_BRANCH_NAME} (tracking remote)"
  fi
elif [[ -n "$BRANCH_EXISTS_LOCALLY" ]]; then
  echo "Release branch '${RELEASE_BRANCH_NAME}' found locally but not on remote '${UPSTREAM_REMOTE_NAME}'."
  MERGE_NEEDED=true
  echo "Switching to local branch '${RELEASE_BRANCH_NAME}'."
  git switch "${RELEASE_BRANCH_NAME}"
  check_command_success "git switch ${RELEASE_BRANCH_NAME}"
  # No reset needed here as there's no remote counterpart to reset to.
  # We'll merge main into this local state.
else
  echo "Release branch '${RELEASE_BRANCH_NAME}' not found locally or on remote."
  echo "Creating new branch '${RELEASE_BRANCH_NAME}' from commit '${TARGET_MAIN_STATE_COMMIT}'."
  git switch -c "${RELEASE_BRANCH_NAME}" "${TARGET_MAIN_STATE_COMMIT}"
  check_command_success "git switch -c ${RELEASE_BRANCH_NAME} ${TARGET_MAIN_STATE_COMMIT}"
  # MERGE_NEEDED remains false as the branch is created directly from the target commit
fi

if [[ "$MERGE_NEEDED" == true ]]; then
  echo "Merging latest from '${MAIN_BRANCH_NAME}' (commit: ${TARGET_MAIN_STATE_COMMIT}) into '${RELEASE_BRANCH_NAME}'..."
  git merge "${TARGET_MAIN_STATE_COMMIT}"
  check_command_success "git merge ${TARGET_MAIN_STATE_COMMIT} into ${RELEASE_BRANCH_NAME}"
  echo "Merge complete. Current status:"
  git status -s # Short status
fi

echo "Current branch is now '${RELEASE_BRANCH_NAME}'."
git --no-pager log -1 --pretty=oneline # Show the commit at the tip of the release branch

# 5. Push the release branch to upstream
echo "5. Pushing branch '${RELEASE_BRANCH_NAME}' to '${UPSTREAM_REMOTE_NAME}'..."
git push -u "${UPSTREAM_REMOTE_NAME}" "${RELEASE_BRANCH_NAME}"
check_command_success "git push -u ${UPSTREAM_REMOTE_NAME} ${RELEASE_BRANCH_NAME}"
echo "Branch '${RELEASE_BRANCH_NAME}' pushed (or updated) on '${UPSTREAM_REMOTE_NAME}'."
echo "Step 4 completed."
echo "--------------------------------------------------"

echo ""
echo "Automation for your Step 4 (push release/<release-series> branch) complete."
echo "Next, you would typically proceed to your Step 5 on your local machine."
echo "You will need to check out the release branch locally and push beta/stable tags."
