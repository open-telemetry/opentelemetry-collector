#!/bin/bash -ex
#
# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0
REF_NAME=v0.0.3/v0.0.1

latest_tag=$(git ls-remote --tags origin | awk -F'/' '{print $NF}' | sort -V | tail -n1)
echo "$latest_tag"

ref_name_regex=$(echo "$REF_NAME" | sed 's/\//\\\//g')
latest_tag_regex=$(echo "$latest_tag" | sed 's/\//\\\//g')
                         
GH_RELEASE=$(gh release create "${REF_NAME}" -t "${REF_NAME}" -n "### Images and binaries here: https://github.com/open-telemetry/opentelemetry-collector-releases/releases/tag/${REF_NAME}

## End User Changelog
$(awk '/## '"$ref_name_regex"'/{flag=1; next} /## '"$latest_tag_regex"'/{flag=0} flag' CHANGELOG.md)

## Go API Changelog
$(awk '/## '"$ref_name_regex"'/{flag=1; next} /## '"$latest_tag_regex"'/{flag=0} flag' CHANGELOG-API.md)

")

if [ "${GH_RELEASE}" != "https://github.com/animeshdas2000/opentelemetry-collector/releases/tag/${REF_NAME}" ]; then
    echo "Error creating ${REF_NAME} release due to: ${GH_RELEASE}"
    exit 1
fi
