// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ocbplugintest

import (
	"golang.org/x/mod/semver"

	"go.opentelemetry.io/collector/cmd/builder/ocbplugin"
)

// IsValidOCBPlugin allows plugin authors to ensure the written plugin
// isn't invalid and can be effectively used in ocbplugin.RunPlugin.
func IsValidOCBPlugin(plugin ocbplugin.OCBPlugin) bool {
	// The plugin must specify a valid MinOCBVersion semver string.
	if !semver.IsValid(plugin.MinOCBVersion()) {
		return false
	}

	return true
}
