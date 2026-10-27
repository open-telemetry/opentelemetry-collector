#!/bin/bash -e
#
# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0

# Compares the size of the otelcorecol binary built from a PR against the
# size of the same binary built from the PR's base ref, and fails if the
# size grew by more than THRESHOLD_PCT.
#
# Usage: check-binary-size.sh <base_size_bytes> <pr_size_bytes> [threshold_pct]

set -u

BASE_SIZE="$1"
PR_SIZE="$2"
THRESHOLD_PCT="${3:-2}"

DIFF=$((PR_SIZE - BASE_SIZE))
PCT=$(awk -v diff="${DIFF}" -v base="${BASE_SIZE}" 'BEGIN { printf "%.2f", (diff / base) * 100 }')

human() {
  awk -v b="$1" 'BEGIN {
    sign = (b < 0) ? "-" : ""
    b = (b < 0) ? -b : b
    split("B KB MB GB", units, " ")
    i = 1
    while (b >= 1024 && i < 4) { b /= 1024; i++ }
    printf "%s%.2f %s", sign, b, units[i]
  }'
}

{
  echo "### otelcorecol binary size (linux/amd64)"
  echo ""
  echo "| | Size |"
  echo "|---|---|"
  echo "| Base | $(human "${BASE_SIZE}") |"
  echo "| PR | $(human "${PR_SIZE}") |"
  echo "| Diff | $(human "${DIFF}") (${PCT}%) |"
} >>"${GITHUB_STEP_SUMMARY:-/dev/stdout}"

echo "Base size: ${BASE_SIZE} bytes"
echo "PR size:   ${PR_SIZE} bytes"
echo "Diff:      ${DIFF} bytes (${PCT}%)"

if awk -v pct="${PCT}" -v th="${THRESHOLD_PCT}" 'BEGIN { exit !(pct > th) }'; then
  echo "::error::otelcorecol binary size grew by ${PCT}%, which exceeds the ${THRESHOLD_PCT}% threshold."
  exit 1
fi
