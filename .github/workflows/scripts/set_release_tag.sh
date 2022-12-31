#!/usr/bin/env bash

TAG="${GITHUB_REF##*/}"
if [[ $TAG =~ ^v[0-9]+\.[0-9]+\.[0-9]+.* ]]
then
    echo "tag=$TAG" >> "$GITHUB_OUTPUT"
fi
