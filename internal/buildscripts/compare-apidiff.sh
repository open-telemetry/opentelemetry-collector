#!/usr/bin/env bash
#
# Copyright The OpenTelemetry Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

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
apidiff_cmd="$(git rev-parse --show-toplevel)/.tools/apidiff"

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
  changes=$(${apidiff_cmd} "$input_dir"/"$package"/apidiff.state "$package")
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
