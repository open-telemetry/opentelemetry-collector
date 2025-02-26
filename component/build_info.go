// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package component // import "go.opentelemetry.io/collector/component"

// BuildInfo is the information that is logged at the application start and
// passed into each component. This information can be overridden in custom build.
type BuildInfo struct {
	// Command is the executable file name, e.g. "otelcol".
	Command string

	// Namespace is the namespace of the collector, e.g. "opentelemetry".
	Namespace string

	// Description is the full name of the collector, e.g. "OpenTelemetry Collector".
	Description string

	// Version string.
	Version string
}

// NewDefaultBuildInfo returns a default BuildInfo.
func NewDefaultBuildInfo() BuildInfo {
	return BuildInfo{
		Command:     "otelcol",
		Namespace:   "opentelemetry",
		Description: "OpenTelemetry Collector",
		Version:     "latest",
	}
}
