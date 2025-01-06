// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scrapertest // import "go.opentelemetry.io/collector/scraper/scrapertest"

import (
	"github.com/google/uuid"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/scraper"
)

var defaultComponentType = component.MustNewType("nop")

// NewNopSettings returns a new nop scraper.Settings.
func NewNopSettings() scraper.Settings {
	return scraper.Settings{
		ID:                component.NewIDWithName(defaultComponentType, uuid.NewString()),
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
		BuildInfo:         component.NewDefaultBuildInfo(),
	}
}
