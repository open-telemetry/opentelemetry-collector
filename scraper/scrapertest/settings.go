// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scrapertest // import "go.opentelemetry.io/collector/scraper/scrapertest"

import (
	"github.com/google/uuid"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/scraper"
)

var NopType = component.MustNewType("nop")

// NewNopSettings returns a new nop scraper.Settings with the given type.
func NewNopSettings(typ component.Type) scraper.Settings {
	return scraper.Settings{
		ID:                component.NewIDWithName(typ, uuid.NewString()),
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
		BuildInfo:         component.NewDefaultBuildInfo(),
	}
}
