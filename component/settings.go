// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package component // import "go.opentelemetry.io/collector/component"

type Settings struct {
	// ID returns the ID of the component that will be created.
	ID ID

	TelemetrySettings

	// BuildInfo can be used by components for informational purposes
	BuildInfo BuildInfo
}
