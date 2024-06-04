// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componenttest // import "go.opentelemetry.io/collector/component/componenttest"

import (
	"github.com/google/uuid"

	"go.opentelemetry.io/collector/component"
)

var nopType = component.MustNewType("nop")

// NewNopSettings returns a new nop settings for Create* functions.
func NewNopSettings() component.Settings {
	return component.Settings{
		ID:                component.NewIDWithName(nopType, uuid.NewString()),
		TelemetrySettings: NewNopTelemetrySettings(),
		BuildInfo:         component.NewDefaultBuildInfo(),
	}
}
