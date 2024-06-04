// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componentexperimental // import "go.opentelemetry.io/collector/componentexperimental"

import "go.opentelemetry.io/collector/component"

func mustNewDataType(strType string) component.DataType {
	return component.MustNewType(strType)
}

// Currently supported data types. Add new data types here when new types are supported in the future.
var (
	// DataTypeProfiles is the data type tag for profiles.
	DataTypeProfiles = mustNewDataType("profiles")
)
