#!/usr/bin/env bash
# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0

# Checks that Go template files (.go.tmpl) use tabs for indentation,
# not spaces. Go source code must be tab-indented (per gofmt), and
# templates that generate Go code should follow the same convention.

set -euo pipefail

failed=0

while IFS= read -r -d '' file; do
    # Find lines that begin with one or more spaces (space-indented).
    # We use grep -n to show line numbers for actionable output.
    if grep -n '^ ' "$file" > /dev/null 2>&1; then
        echo "ERROR: $file contains space indentation (expected tabs):"
        grep -n '^ ' "$file" | head -10
        echo ""
        failed=1
    fi
done < <(find . -name '*.go.tmpl' -type f -print0)

if [ "$failed" -eq 1 ]; then
    echo "FAILED: Some Go template files use spaces for indentation instead of tabs."
    echo "Please convert all leading spaces to tabs in the listed files."
    exit 1
fi

echo "OK: All Go template files use tab indentation."
