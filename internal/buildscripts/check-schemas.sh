#!/usr/bin/env bash
# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0
#
# check-schemas.sh — assert every Go `mapstructure` config field has a
# corresponding entry in the neighbouring config.schema.yaml.
#
# This is intentionally a field-PRESENCE check, not a regenerate-and-diff
# check: schemas in this repo are bootstrapped by the contrib schemagen
# tool but may carry hand-edits (richer descriptions, enums, validation
# notes) that a fresh regeneration would overwrite. We catch the realistic
# drift case (a new Go field landed without a schema update) without
# fighting acceptable manual edits.
#
# A schema "contains" a tag if the tag name appears as a key under any
# `properties` block anywhere in the document. That covers top-level
# properties as well as nested $defs.
#
# Tags with the special value `,squash` are skipped — squashed embeds
# don't introduce a property; their fields flatten into the parent and
# are validated via the schema's `allOf` $ref to the embedded type's
# schema (verified transitively when that schema itself is checked).

set -euo pipefail

if ! command -v yq >/dev/null 2>&1; then
    echo "check-schemas: yq is required but not installed" >&2
    exit 2
fi

root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
cd "$root"

failed=0

while IFS= read -r schema; do
    dir=$(dirname "$schema")

    # Collect every key that lives under a `properties` block, at any depth.
    # `select(has("properties"))` finds every map node that has a properties
    # child; we then take that child's keys.
    props=$(yq '[.. | select(has("properties")) | .properties | keys[]] | unique | .[]' "$schema" 2>/dev/null | sort -u)

    # Extract `mapstructure:"NAME..."` tag names from every non-test .go
    # file in the same directory. Strip the `,squash` / `,omitempty`
    # suffixes and drop any tag whose name is empty (what `,squash` looks
    # like). Test files are excluded — they may declare throwaway structs
    # whose fields are not part of the component's public config surface.
    tags=$(find "$dir" -maxdepth 1 -name '*.go' -not -name '*_test.go' -print0 \
        | xargs -0 grep -hE 'mapstructure:"[^"]+' 2>/dev/null \
        | sed -nE 's/.*mapstructure:"([^,"]*).*/\1/p' \
        | awk 'NF' \
        | sort -u || true)

    [[ -z "$tags" ]] && continue

    while IFS= read -r tag; do
        if ! grep -qFx -- "$tag" <<<"$props"; then
            echo "DRIFT: $schema is missing property '$tag' (mapstructure tag found in $dir)"
            failed=1
        fi
    done <<<"$tags"
done < <(find . \
    -path "*/testdata/*" -prune -o \
    -path "*/internal/metadata/*" -prune -o \
    -path "*/cmd/schemagen/*" -prune -o \
    -path "*/cmd/mdatagen/*" -prune -o \
    -name "config.schema.yaml" -print)

if [[ $failed -ne 0 ]]; then
    echo
    echo "One or more config.schema.yaml files are missing properties that"
    echo "exist as mapstructure-tagged fields in the corresponding Go struct."
    echo "Either regenerate via 'make generate-schemas' and re-apply any"
    echo "needed hand-edits, or add the missing properties by hand."
    exit 1
fi

echo "check-schemas: OK"
