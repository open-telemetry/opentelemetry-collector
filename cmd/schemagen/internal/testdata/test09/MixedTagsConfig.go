// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package test09

// MixedTagsConfig showcases json and mapstructure tag interactions.
type MixedTagsConfig struct {
	// PrimaryEndpoint uses both json and mapstructure tags.
	PrimaryEndpoint string `json:"primary_endpoint" mapstructure:"primary"`
	// LegacyEndpoint relies on mapstructure only.
	LegacyEndpoint string `mapstructure:"legacy_endpoint"`
	// JSONOnlyFlag is only tagged with json.
	JSONOnlyFlag bool `json:"json_only"`
	// MapOnlyCounter demonstrates mapstructure tags with modifiers.
	MapOnlyCounter int `mapstructure:"map_only,omitempty"`
	// GoFallbackField has no tags and keeps the Go field name.
	GoFallbackField string
}
