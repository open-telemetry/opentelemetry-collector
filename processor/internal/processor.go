// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/processor/internal"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata"
)

// Settings is passed to Create* functions in Factory.
type Settings struct {
	// ID returns the ID of the component that will be created.
	ID component.ID

	component.TelemetrySettings

	// BuildInfo can be used by components for informational purposes
	BuildInfo component.BuildInfo

	// Publishers can be used by components to publish data after processing
	Publishers []pdata.Publisher
}
