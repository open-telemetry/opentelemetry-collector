#!/usr/bin/env bash

files=(
    bin/otelcol_darwin_amd64
    bin/otelcol_darwin_arm64
    bin/otelcol_linux_arm64
    bin/otelcol_linux_ppc64le
    bin/otelcol_linux_amd64
    bin/otelcol_windows_amd64.exe
);
for f in "${files[@]}"
do
    if [[ ! -f $f ]]
    then
        echo "$f does not exist."
        echo "passed=false" >> "$GITHUB_OUTPUT"
        exit 0 
    fi
done
echo "passed=true" >> "$GITHUB_OUTPUT"
