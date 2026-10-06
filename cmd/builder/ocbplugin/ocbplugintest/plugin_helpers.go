package ocbplugintest

import (
	"go.opentelemetry.io/collector/cmd/builder/ocbplugin"
	"golang.org/x/mod/semver"
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
