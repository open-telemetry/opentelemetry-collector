// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componentprofiles // import "go.opentelemetry.io/collector/component/componentprofiles"

import "go.opentelemetry.io/collector/component"

func mustNewDataType(strType string) component.DataType {
	return component.MustNewType(strType)
}

var (
	// DataTypeProfiles is the data type tag for profiles.
	DataTypeProfiles = mustNewDataType("profiles")
)
