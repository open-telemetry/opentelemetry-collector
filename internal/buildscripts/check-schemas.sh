#!/usr/bin/env bash
# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0
#
# check-schemas.sh — assert every committed component config.schema.yaml is
# in sync with its Go config struct by regenerating into a temp tree and
# diffing the property-name set against the committed schema.
#
# Why a property-set diff instead of regenerate-and-byte-diff: schemas in
# this repo carry hand-edits (richer descriptions, validation notes, enum
# constraints) that a fresh regeneration would overwrite. We want to catch
# the realistic drift case — a new Go field landed without a schema update,
# or vice versa — without fighting acceptable manual edits. The property
# names produced by schemagen are derived directly from `mapstructure`
# tags on the Go struct, so comparing the regenerated set against the
# committed set catches additions, removals, and renames.
#
# Scope: only schemas under the dirs listed in CHECKED_DIRS. Shared-library
# schemas (config/*, scraper/scraperhelper, exporter/exporterhelper, etc.)
# have a separate maintenance lifecycle and are out of scope for this check.
# To grow the scope, add the new dir to CHECKED_DIRS below.

set -euo pipefail

if ! command -v yq >/dev/null 2>&1; then
    echo "check-schemas: yq is required but not installed" >&2
    exit 2
fi

root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
cd "$root"

tmp=$(mktemp -d -t check-schemas.XXXXXX)
trap 'rm -rf "$tmp"' EXIT

CHECKED_DIRS=(
    "exporter/debugexporter"
    "exporter/otlpexporter"
    "exporter/otlphttpexporter"
    "receiver/otlpreceiver"
    "processor/batchprocessor"
    "processor/memorylimiterprocessor"
    "extension/memorylimiterextension"
    "extension/zpagesextension"
    "internal/memorylimiter"
)

# Yield the keys of every map node under any `properties` block, at any
# depth. This is the property name set that schemagen derives from the
# Go struct's `mapstructure` tags. Descriptions/enums/$refs are ignored.
property_keys() {
    yq '[.. | select(has("properties")) | .properties | keys[]] | unique | .[]' "$1" 2>/dev/null | sort -u
}

failed=0

for dir in "${CHECKED_DIRS[@]}"; do
    schema="$root/$dir/config.schema.yaml"
    if [[ ! -f "$schema" ]]; then
        echo "DRIFT: $dir/config.schema.yaml missing — expected to exist (listed in CHECKED_DIRS)"
        failed=1
        continue
    fi

    out="$tmp/$(echo "$dir" | tr / _)"
    mkdir -p "$out"

    # Regenerate the schema for $dir into $out using the in-tree schemagen.
    if ! (cd "$root/cmd/schemagen" && go run . -o "$out" "$root/$dir" >/dev/null 2>&1); then
        echo "DRIFT: $dir — schemagen failed to regenerate"
        failed=1
        continue
    fi

    regenerated="$out/config.schema.yaml"
    if [[ ! -f "$regenerated" ]]; then
        echo "DRIFT: $dir — schemagen did not produce config.schema.yaml"
        failed=1
        continue
    fi

    if ! diff_output=$(diff <(property_keys "$schema") <(property_keys "$regenerated")); then
        echo "DRIFT: $dir/config.schema.yaml property set differs from regenerated schema:"
        echo "    ${diff_output//$'\n'/$'\n    '}"
        echo "    (< committed, > regenerated)"
        failed=1
    fi
done

if [[ $failed -ne 0 ]]; then
    echo
    echo "One or more config.schema.yaml files have drifted from the Go config struct."
    echo "Run 'make generate-schemas' to refresh, re-apply any needed hand-edits to"
    echo "descriptions/enums/validation notes, and commit the result."
    exit 1
fi

echo "check-schemas: OK"
