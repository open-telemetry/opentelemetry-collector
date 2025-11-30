#!/usr/bin/env bash
#
# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0

# This script is used to create API state snapshots used to validate releases are not breaking backwards compatibility.

usage() {
  echo "Usage: $0"
  echo
  echo "-d  Dry-run mode. No project files will not be modified. Default: 'false'"
  echo "-p  Package to generate API state snapshot of. Default: ''"
  echo "-o  Output directory where state will be written to. Default: './internal/data/apidiff'"
  exit 1
}

dry_run=false
package=""
output_dir="./internal/data/apidiff"
repo_toplevel="$( git rev-parse --show-toplevel )"
tools_mod_file="${repo_toplevel}/internal/tools/go.mod"

while getopts "dp:o:" o; do
    case "${o}" in
        d)
            dry_run=true
            ;;
        p)
            package=$OPTARG
            ;;
        o)
            output_dir=$OPTARG
            ;;
        *)
            usage
            ;;
    esac
done
shift $((OPTIND-1))

if [ -z "$package" ]; then
  usage
fi

set -ex

# Create temp dir for generated files.
# Source: https://unix.stackexchange.com/a/84980
tmp_dir=$(mktemp -d 2>/dev/null || mktemp -d -t 'apidiff')
clean_up() {
    ARG=$?
    if [ $dry_run = true ]; then
      echo "Dry-run complete. Generated files can be found in $tmp_dir"
    else
      rm -rf "$tmp_dir"
    fi
    exit $ARG
}
trap clean_up EXIT

mkdir -p "$tmp_dir/$package"

go tool -modfile "${tools_mod_file}" apidiff -w "$tmp_dir"/"$package"/apidiff.state "$package"

# Copy files if not in dry-run mode.
if [ $dry_run = false ]; then
  mkdir -p "$output_dir/$package" && \
  cp "$tmp_dir/$package/apidiff.state" \
     "$output_dir/$package"
fi
