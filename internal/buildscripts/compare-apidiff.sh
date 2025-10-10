#!/usr/bin/env bash
#
# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0

# This script is used to compare API state snapshots to the current package state in order to validate releases are not breaking backwards compatibility.

usage() {
  echo "Usage: $0"
  echo
  echo "-c  Check-incompatibility mode. Script will fail if an incompatible change is found. Default: 'false'"
  echo "-p  Package to generate API state snapshot of. Default: ''"
  echo "-d  directory where prior states will be read from. Default: './internal/data/apidiff'"
  exit 1
}

package=""
input_dir="./internal/data/apidiff"
check_only=false
repo_toplevel="$( git rev-parse --show-toplevel )"
tools_mod_file="${repo_toplevel}/internal/tools/go.mod"

while getopts "cp:d:" o; do
    case "${o}" in
        c)
            check_only=true
            ;;
        p)
            package=$OPTARG
            ;;
        d)
            input_dir=$OPTARG
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

set -e

if [ -e "$input_dir"/"$package"/apidiff.state ]; then
  changes=$(go tool -modfile "${tools_mod_file}" apidiff "$input_dir"/"$package"/apidiff.state "$package")
  if [ -n "$changes" ] && [ "$changes" != " " ]; then
    SUB='Incompatible changes:'
    if [ $check_only = true ] && [[ "$changes" =~ .*"$SUB".* ]]; then
      echo "Incompatible Changes Found."
      echo "Check the logs in the GitHub Action log group: 'Compare-States'."
      exit 1
    else
      echo "Changes found in $package:"
      echo "$changes"
    fi
  fi
fi
