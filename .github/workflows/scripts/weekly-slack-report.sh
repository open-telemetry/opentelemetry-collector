#!/usr/bin/env bash
#
# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0
#
# Posts a summary of open PRs with specific labels to Slack.
#
# The Slack workflow is named "OTel Collector Weekly summary" (ID: Wf0AUBDM76KH).
# opentelemetry-collector maintainers are workflow managers for it.
#
# IMPORTANT NOTE: Changes to this script need changes to the "Data Variables"
# definition of the Slack webhook.
# Current defined variables are:
# - "opentelemetry_collector_ready_to_merge_url"
# - "opentelemetry_collector_ready_to_merge_count"
# - "opentelemetry_collector_final_comment_period_url"
# - "opentelemetry_collector_final_comment_period_count"


set -euo pipefail

missing=()
[[ -z "${REPO:-}" ]] && missing+=("REPO")
[[ -z "${GITHUB_TOKEN:-}" ]] && missing+=("GITHUB_TOKEN")
[[ -z "${SLACK_WEBHOOK_URL:-}" ]] && missing+=("SLACK_WEBHOOK_URL")
if (( ${#missing[@]} > 0 )); then
    echo "Required env vars not set: ${missing[*]}"
    exit 1
fi

declare -A LABELS=(
    ["ready-to-merge"]="ready_to_merge"
    ["rfc:final-comment-period"]="final_comment_period"
)

repo_prefix="${REPO##*/}"
repo_prefix="${repo_prefix//-/_}"

payload='{}'
for label in "${!LABELS[@]}"; do
    prefix="${repo_prefix}_${LABELS[$label]}"
    q="is:pr is:open repo:${REPO} label:\"${label}\""
    n=$(gh api -X GET search/issues -f q="${q}" --jq '.total_count')
    url="https://github.com/${REPO}/pulls?q=$(jq -rn --arg q "${q}" '$q|@uri')"
    payload=$(jq \
        --arg count_key "${prefix}_count" \
        --arg count_val "${n}" \
        --arg url_key "${prefix}_url" \
        --arg url_val "${url}" \
        '. + {($count_key): $count_val, ($url_key): $url_val}' <<<"${payload}")
done

curl -fsS -X POST -H 'Content-Type: application/json' \
    --data-binary @- "${SLACK_WEBHOOK_URL}" <<<"${payload}"
