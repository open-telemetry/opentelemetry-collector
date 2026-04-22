#!/usr/bin/env bash
#
# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0
#
# Posts a summary of open PRs with specific labels to Slack.
#
# The Slack payload uses the rich_text block so bullets render as real list
# items. See https://docs.slack.dev/reference/block-kit/blocks/rich-text-block
# for the schema, and https://app.slack.com/block-kit-builder to preview
# a payload.

set -euo pipefail

missing=()
[[ -z "${REPO:-}" ]] && missing+=("REPO")
[[ -z "${GITHUB_TOKEN:-}" ]] && missing+=("GITHUB_TOKEN")
[[ -z "${SLACK_WEBHOOK_URL:-}" ]] && missing+=("SLACK_WEBHOOK_URL")
if (( ${#missing[@]} > 0 )); then
    echo "Required env vars not set: ${missing[*]}"
    exit 1
fi

LABELS=(
    "ready-to-merge"
    "rfc:final-comment-period"
)

items='[]'
for label in "${LABELS[@]}"; do
    q="is:pr is:open repo:${REPO} label:\"${label}\""
    n=$(gh api -X GET search/issues -f q="${q}" --jq '.total_count')
    if [[ "${n:-0}" -eq 0 ]]; then
        echo "Skipping ${label}: no open PRs."
        continue
    fi
    url="https://github.com/${REPO}/pulls?q=$(jq -rn --arg q "${q}" '$q|@uri')"
    items=$(jq \
        --arg lbl "${label}" \
        --argjson n "${n}" \
        --arg url "${url}" \
        '. + [{
           type: "rich_text_section",
           elements: [
             { type: "text", text: "PRs with label " },
             { type: "text", text: $lbl, style: { code: true } },
             { type: "text", text: " (\($n)): " },
             { type: "link", url: $url, text: "list" }
           ]
         }]' <<<"${items}")
done

if [[ "$(jq 'length' <<<"${items}")" -eq 0 ]]; then
    echo "No PRs match any tracked label; skipping Slack post."
    exit 0
fi

payload=$(jq -n --argjson items "${items}" --arg repo "${REPO##*/}" '{
    blocks: [{
        type: "rich_text",
        elements: [
            {
                type: "rich_text_section",
                elements: [
                    { type: "text",
                      text: "Weekly PR summary for \($repo)",
                      style: { bold: true } }
                ]
            },
            {
                type: "rich_text_list",
                style: "bullet",
                elements: $items
            }
        ]
    }]
}')

curl -fsS -X POST -H 'Content-Type: application/json' \
    --data-binary @- "${SLACK_WEBHOOK_URL}" <<<"${payload}"
