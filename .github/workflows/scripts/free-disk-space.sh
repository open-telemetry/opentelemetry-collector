#!/usr/bin/env bash
#
# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0

echo "Available disk space before:"
df -h /

# The Android SDK is the biggest culprit for the lack of disk space in CI.
# It is installed into /usr/local/lib/android manually (ie. not with apt) by this script:
# https://github.com/actions/runner-images/blob/main/images/ubuntu/scripts/build/install-android-sdk.sh
# so let's delete the directory manually.
echo "Deleting unused Android SDK and tools..."
sudo rm -rf /usr/local/lib/android

echo "Available disk space after:"
df -h /
