#!/usr/bin/env bash

# This script is used to compare API state snapshots to the current package state in order to validate releases are not breaking backwards compatibility.

usage() {
  echo "Usage: $0"
  echo
  echo "-p  Package to generate API state snapshot of. Default: ''"
  echo "-d  directory where prior states will be read from. Default: './internal/data/apidiff'"
  exit 1
}

package=""
input_dir="./internal/data/apidiff"


while getopts "p:d:" o; do
    case "${o}" in
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

if [ -z $package ]; then
  usage
fi

set -e

if [ -d $input_dir/$package ]; then
  changes=$(apidiff $input_dir/$package/apidiff.state $package)
  if [ ! -z "$changes" -a "$changes"!=" " ]; then
    echo "Changes found in $package:"
    echo "$changes"
  fi
fi