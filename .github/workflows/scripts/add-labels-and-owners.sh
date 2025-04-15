#!/usr/bin/env bash
#
# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0
#
# Adds code owners without write access as reviewers on a PR. Note that
# the code owners must still be a member of the `open-telemetry`
# organization.
#
# Note that since this script is considered a requirement for PRs,
# it should never fail.

set -euo pipefail

if [[ -z "${REPO:-}" || -z "${PR:-}" ]]; then
    echo "One or more of REPO and PR have not been set, please ensure each is set."
    exit 0
fi

main () {
    CUR_DIRECTORY=$(dirname "$0")
    # Reviews may have comments that need to be cleaned up for jq,
    # so restrict output to only printable characters and ensure escape
    # sequences are removed.
    # The latestReviews key only returns the latest review for each reviewer,
    # cutting out any other reviews. We use that instead of requestedReviews
    # since we need to get the list of users eligible for requesting another
    # review. The GitHub CLI does not offer a list of all reviewers, which
    # is only available through the API. To cut down on API calls to GitHub,
    # we use the latest reviews to determine which users to filter out.
    JSON=$(gh pr view "${PR}" --json "files,author,latestReviews" | LC_ALL=C tr -dc '[:print:]' | sed -E 's/\\[a-z]//g')
    AUTHOR=$(echo -n "${JSON}"| jq -r '.author.login')
    FILES=$(echo -n "${JSON}"| jq -r '.files[].path')
    REVIEW_LOGINS=$(echo -n "${JSON}"| jq -r '.latestReviews[].author.login')
    COMPONENTS=$(bash "${CUR_DIRECTORY}/get-components.sh")
    REVIEWERS=""
    LABELS=""
    declare -A PROCESSED_COMPONENTS
    declare -A REVIEWED

    for REVIEWER in ${REVIEW_LOGINS}; do
        # GitHub adds "app/" in front of user logins. The API docs don't make
        # it clear what this means or whether it will always be present. The
        # '/' character isn't a valid character for usernames, so this won't
        # replace characters within a username.
        REVIEWED["@${REVIEWER//app\//}"]=true
    done

    if [[ -v REVIEWED[@] ]]; then
        echo "Users that have already reviewed this PR and will not have another review requested:" "${!REVIEWED[@]}"
    else
        echo "This PR has not yet been reviewed, all code owners are eligible for a review request"
    fi

    RISKY_REGEX='
     ^.github/workflows/prepare-release.yml$
    |^.github/workflows/scripts/release-prepare-release.sh$
    |^Makefile$
    |^Makefile.Common$
    '
    RISKY_REGEX="$(echo "$RISKY_REGEX" | tr -d ' \n')"
    RISKY_FILES="$(echo "$FILES" | grep -E "$RISKY_REGEX")"
    if [[ -n "${RISKY_FILES}" ]]; then
        echo "This PR may affect the release process, as it touches the following files:" \
            "$(echo "$RISKY_FILES" | sed -E 's/\n/, /')"
        LABELS="release:risky-change"
    else
        echo "This PR does not have release-affecting changes."
    fi

    for COMPONENT in ${COMPONENTS}; do
        # Files will be in alphabetical order and there are many files to
        # a component, so loop through files in an inner loop. This allows
        # us to remove all files for a component from the list so they
        # won't be checked against the remaining components in the components
        # list. This provides a meaningful speedup in practice.
        for FILE in ${FILES}; do
            MATCH=$(echo -n "${FILE}" | grep -E "^${COMPONENT}" || true)

            if [[ -z "${MATCH}" ]]; then
                continue
            fi

            # If we match a file with a component we don't need to process the file again.
            FILES=$(echo -n "${FILES}" | grep -v "${FILE}")

            if [[ -v PROCESSED_COMPONENTS["${COMPONENT}"] ]]; then
                continue
            fi

            PROCESSED_COMPONENTS["${COMPONENT}"]=true

            OWNERS=$(COMPONENT="${COMPONENT}" bash "${CUR_DIRECTORY}/get-codeowners.sh")

            for OWNER in ${OWNERS}; do
                # Users that leave reviews are removed from the "requested reviewers"
                # list and are eligible to have another review requested. We only want
                # to request a review once, so remove them from the list.
                if [[ -v REVIEWED["${OWNER}"] || "${OWNER}" = "@${AUTHOR}" ]]; then
                    continue
                fi

                if [[ -n "${REVIEWERS}" ]]; then
                    REVIEWERS+=","
                fi
                REVIEWERS+=$(echo -n "${OWNER}" | sed -E 's/@(.+)/"\1"/')
            done

            # Convert the CODEOWNERS entry to a label
            COMPONENT_NAME=$(echo -n "${COMPONENT}" | sed -E 's%^(.+)/(.+)\1%\1/\2%')

            if (( "${#COMPONENT_NAME}" > 50 )); then
                echo "'${COMPONENT_NAME}' exceeds GitHub's 50-character limit on labels, skipping adding label"
                continue
            fi

            if [[ -n "${LABELS}" ]]; then
                LABELS+=","
            fi
            LABELS+="${COMPONENT_NAME}"
        done
    done

    if [[ -n "${LABELS}" ]]; then
        echo "Adding labels: ${LABELS}"
        gh pr edit "${PR}" --add-label "${LABELS}" || echo "Failed to add labels"
    else
        echo "No labels found"
    fi

    # Note that adding the labels above will not trigger any other workflows to
    # add code owners, so we have to do it here.
    #
    # We have to use the GitHub API directly due to an issue with how the CLI
    # handles PR updates that causes it require access to organization teams,
    # and the GitHub token doesn't provide that permission.
    # For more: https://github.com/cli/cli/issues/4844
    #
    # The GitHub API validates that authors are not requested to review, but
    # accepts duplicate logins and logins that are already reviewers.
    if [[ -n "${REVIEWERS}" ]]; then
        echo "Requesting review from ${REVIEWERS}"
        curl \
            -X POST \
            -H "Accept: application/vnd.github+json" \
            -H "Authorization: Bearer ${GITHUB_TOKEN}" \
            "https://api.github.com/repos/${REPO}/pulls/${PR}/requested_reviewers" \
            -d "{\"reviewers\":[${REVIEWERS}]}" \
            | jq ".message" \
            || echo "jq was unable to parse GitHub's response"
    else
        echo "No code owners found"
    fi
}

# We don't want this workflow to ever fail and block a PR,
# so ensure all errors are caught.
main || echo "Failed to run $0"
