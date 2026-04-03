#!/bin/bash -e
#
# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0
#
# This script updates all go.opentelemetry.io/collector dependencies in go.mod
# files to use pseudo-versions derived from a specific commit on the main branch.
#
# This is useful for ensuring that all intra-core dependencies are consistent
# when downstream repos (like collector-contrib) update to the latest main.
#
# Usage:
#   ./update-core-pseudo-versions.sh [--commit <sha>] [--target-dir <dir>]
#
# Options:
#   --commit <sha>       The commit SHA to use for pseudo-versions.
#                        Defaults to HEAD of the main branch.
#   --target-dir <dir>   The directory containing go.mod files to update.
#                        Defaults to the current directory. When used from
#                        collector-contrib, set this to the contrib repo root.
#
# Environment variables:
#   CORE_REPO_DIR        Path to a local checkout of opentelemetry-collector.
#                        If not set, the script expects to be run from inside
#                        the core repo (or uses the repo where this script lives).

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="${CORE_REPO_DIR:-$(cd "${SCRIPT_DIR}/../../.." && pwd)}"

COMMIT=""
TARGET_DIR="."

while [[ $# -gt 0 ]]; do
    case "$1" in
        --commit)
            COMMIT="$2"
            shift 2
            ;;
        --target-dir)
            TARGET_DIR="$2"
            shift 2
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Resolve target directory to absolute path
TARGET_DIR="$(cd "${TARGET_DIR}" && pwd)"

# If no commit specified, use HEAD of main
if [[ -z "${COMMIT}" ]]; then
    pushd "${REPO_ROOT}" > /dev/null
    COMMIT="$(git rev-parse HEAD)"
    popd > /dev/null
    echo "Using HEAD commit: ${COMMIT}"
fi

# Get the short commit hash (first 12 characters)
SHORT_COMMIT="${COMMIT:0:12}"

# Get the commit timestamp in the format Go uses for pseudo-versions (YYYYMMDDHHmmss)
pushd "${REPO_ROOT}" > /dev/null
COMMIT_TIMESTAMP="$(git log -1 --format='%cd' --date=format:'%Y%m%d%H%M%S' "${COMMIT}")"
popd > /dev/null

echo "Commit: ${COMMIT}"
echo "Timestamp: ${COMMIT_TIMESTAMP}"
echo ""

# Parse versions.yaml to extract module sets and their versions
# We need to compute pseudo-versions for both "stable" and "beta" module sets.
VERSIONS_YAML="${REPO_ROOT}/versions.yaml"
if [[ ! -f "${VERSIONS_YAML}" ]]; then
    echo "Error: versions.yaml not found at ${VERSIONS_YAML}"
    exit 1
fi

# Extract version for a given module set from versions.yaml
get_version_for_set() {
    local set_name="$1"
    # Parse the version field under the given module-set
    awk -v set="${set_name}" '
        /^  [a-z]+:/ { current_set = $1; gsub(/:/, "", current_set) }
        current_set == set && /version:/ { print $2; exit }
    ' "${VERSIONS_YAML}"
}

# Extract modules for a given module set from versions.yaml
get_modules_for_set() {
    local set_name="$1"
    awk -v set="${set_name}" '
        /^  [a-z]+:/ { current_set = $1; gsub(/:/, "", current_set) }
        current_set == set && /^      - / { gsub(/^      - /, ""); print }
    ' "${VERSIONS_YAML}"
}

STABLE_VERSION="$(get_version_for_set "stable")"
BETA_VERSION="$(get_version_for_set "beta")"

if [[ -z "${STABLE_VERSION}" ]] || [[ -z "${BETA_VERSION}" ]]; then
    echo "Error: Could not extract versions from versions.yaml"
    echo "  stable: ${STABLE_VERSION}"
    echo "  beta: ${BETA_VERSION}"
    exit 1
fi

echo "Current stable version: ${STABLE_VERSION}"
echo "Current beta version: ${BETA_VERSION}"

# Compute pseudo-versions
# Go pseudo-version format: vX.Y.(Z+1)-0.YYYYMMDDHHmmss-abcdefabcdef
# For pre-release versions (v0.x.y), the format is the same.
compute_pseudo_version() {
    local version="$1"
    # Strip leading 'v'
    local ver="${version#v}"
    # Split into major.minor.patch
    local major minor patch
    IFS='.' read -r major minor patch <<< "${ver}"
    # Increment patch
    patch=$((patch + 1))
    echo "v${major}.${minor}.${patch}-0.${COMMIT_TIMESTAMP}-${SHORT_COMMIT}"
}

STABLE_PSEUDO="$(compute_pseudo_version "${STABLE_VERSION}")"
BETA_PSEUDO="$(compute_pseudo_version "${BETA_VERSION}")"

echo "Stable pseudo-version: ${STABLE_PSEUDO}"
echo "Beta pseudo-version: ${BETA_PSEUDO}"
echo ""

# Build a mapping of module -> pseudo-version
declare -A MODULE_VERSIONS

while IFS= read -r module; do
    MODULE_VERSIONS["${module}"]="${STABLE_PSEUDO}"
done <<< "$(get_modules_for_set "stable")"

while IFS= read -r module; do
    MODULE_VERSIONS["${module}"]="${BETA_PSEUDO}"
done <<< "$(get_modules_for_set "beta")"

echo "Found ${#MODULE_VERSIONS[@]} core modules to update"
echo ""

# Find all go.mod files in the target directory
GO_MOD_FILES="$(find "${TARGET_DIR}" -name "go.mod" -not -path "*/vendor/*" | sort)"

UPDATED_COUNT=0

for go_mod in ${GO_MOD_FILES}; do
    # Get the module name of this go.mod
    MOD_NAME="$(head -1 "${go_mod}" | awk '{print $2}')"

    EDITS=""
    for module in "${!MODULE_VERSIONS[@]}"; do
        pseudo="${MODULE_VERSIONS[${module}]}"

        # Skip self-references
        if [[ "${MOD_NAME}" == "${module}" ]]; then
            continue
        fi

        # Check if this go.mod has a replace directive for this module pointing to a local path.
        # If so, skip it — local replaces take precedence and the version doesn't matter.
        if grep -qE "^replace\s+${module//./\\.}\s+=>" "${go_mod}" 2>/dev/null; then
            continue
        fi

        # Check if this module is referenced in require blocks (direct or indirect)
        if grep -qE "^\s+${module//./\\.}\s+v" "${go_mod}" 2>/dev/null; then
            EDITS+=" -require=${module}@${pseudo}"
        fi
    done

    if [[ -n "${EDITS}" ]]; then
        echo "Updating: ${go_mod}"
        # shellcheck disable=SC2086
        go mod edit ${EDITS} "${go_mod}"
        UPDATED_COUNT=$((UPDATED_COUNT + 1))
    fi
done

echo ""
echo "Updated ${UPDATED_COUNT} go.mod files"

# Run go mod tidy on all updated modules if requested
if [[ "${UPDATED_COUNT}" -gt 0 ]]; then
    echo ""
    echo "Running 'go mod tidy' across all modules..."
    pushd "${TARGET_DIR}" > /dev/null
    # If a Makefile with gotidy target exists, use it
    if [[ -f "Makefile" ]] && grep -q "^gotidy:" "Makefile"; then
        make gotidy
    else
        # Otherwise, tidy each go.mod individually
        for go_mod in ${GO_MOD_FILES}; do
            mod_dir="$(dirname "${go_mod}")"
            echo "  Tidying ${mod_dir}"
            (cd "${mod_dir}" && go mod tidy -compat=1.25.0 2>/dev/null || true)
        done
    fi
    popd > /dev/null
fi

echo ""
echo "Done! All intra-core dependencies updated to pseudo-versions."
echo "  Stable modules: ${STABLE_PSEUDO}"
echo "  Beta modules:   ${BETA_PSEUDO}"
