#!/bin/bash

set -exuo pipefail

CIRCLE_PROJECT_USERNAME="${CIRCLE_PROJECT_USERNAME:-open-telemetry}"
CIRCLE_PROJECT="${CIRCLE_PROJECT:-opentelemetry-collector}"
GOOS="${GOOS:-linux}"
GOARCH="${GOARCH:-amd64}"
CIRCLE_JOB="${CIRCLE_JOB:-goweight-otelcol_${GOOS}_${GOARCH}}"
API_URL="https://circleci.com/api/v1.1/project/github/${CIRCLE_PROJECT_USERNAME}/${CIRCLE_PROJECT}"

max_offset=1000
offset=0
build_num=""
while [[ -z "$build_num" || "$build_num" = "null" ]] && [[ $offset -le $max_offset ]]; do
    build_num=$( curl -s "${API_URL}?filter=successful&limit=100&offset=${offset}" | jq -r "[.[] | select(.branch == \"master\" and .workflows.job_name == \"$CIRCLE_JOB\")][0] | .build_num" )
    (( offset += 100 ))
    sleep 2
done

if [[ -z "$build_num" || "$build_num" = "null" ]]; then
    echo "Unable to get latest build number for master branch job $CIRCLE_JOB from last $max_offset builds"
    exit 0
fi

url=$( curl -s "${API_URL}/${build_num}/artifacts" | jq -r '.[0] | .url' )
if [[ -z "$url" || "$url" = "null" ]]; then
    echo "Unable to get artifact url for build $build_num"
    exit 0
fi

mkdir -p ~/goweight
curl -sSL "$url" > ~/goweight/${CIRCLE_JOB}.json || true
