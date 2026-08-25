#!/usr/bin/env sh
#
# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0
#
# Get a list of components within the repository that have some form of ownership
# ascribed to them.

grep -E '^[A-Za-z0-9/]' .github/CODEOWNERS | \
    awk '{ print $1 }' | \
    sed -E 's%(.+)/$%\1%'
