// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ocbplugintest

import (
	"go.opentelemetry.io/collector/cmd/builder/ocbplugin"
	"go.opentelemetry.io/collector/cmd/builder/ocbplugin/internal/helper"
)

// IsValidOCBPlugin allows plugin authors to ensure the written plugin
// isn't invalid and can be effectively used in ocbplugin.RunPlugin.
func IsValidOCBPlugin(plugin ocbplugin.OCBPlugin) error {
	// The plugin must specify a valid MinOCBVersion semver string.
	min := plugin.MinOCBVersion()

	if _, err := helper.NormalizeAndValidateSemver(min); err != nil {
		return err
	}

	return nil
}
