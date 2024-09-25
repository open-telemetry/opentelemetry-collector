// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componentprofiles // import "go.opentelemetry.io/collector/component/componentprofiles"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/internal/globalsignal"
)

// nolint
func mustNewDataType(strType string) component.DataType {
	return component.MustNewType(strType)
}

var (
	// DataTypeProfiles is the data type tag for profiles.
	//
	// Deprecated: [v0.110.0] Use SignalProfiles instead
	DataTypeProfiles = mustNewDataType("profiles")

	SignalProfiles = globalsignal.MustNewSignal("profiles")
)
