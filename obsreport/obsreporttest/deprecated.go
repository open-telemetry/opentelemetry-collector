// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package obsreporttest // import "go.opentelemetry.io/collector/obsreport/obsreporttest"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
)

// Deprecated: [0.93.0] Use componenttest.TestTelemetry instead
type TestTelemetry = componenttest.TestTelemetry

// SetupTelemetry does setup the testing environment to check the metrics recorded by receivers, producers or exporters.
// The caller must pass the ID of the component that intends to test, so the CreateSettings and Check methods will use.
// The caller should defer a call to Shutdown the returned TestTelemetry.
//
// Deprecated: [0.93.0] Use componenttest.SetupTelemetry instead
func SetupTelemetry(id component.ID) (TestTelemetry, error) {
	return componenttest.SetupTelemetry(id)
}

// CheckScraperMetrics checks that for the current exported values for metrics scraper metrics match given values.
// When this function is called it is required to also call SetupTelemetry as first thing.
//
// Deprecated: [0.93.0] Use TestTelemetry.CheckScraperMetrics instead
func CheckScraperMetrics(tts TestTelemetry, receiver component.ID, scraper component.ID, scrapedMetricPoints, erroredMetricPoints int64) error {
	return tts.CheckScraperMetrics(receiver, scraper, scrapedMetricPoints, erroredMetricPoints)
}
