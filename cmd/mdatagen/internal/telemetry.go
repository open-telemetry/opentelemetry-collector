// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/cmd/mdatagen/internal"

type Telemetry struct {
	Metrics map[MetricName]Metric `mapstructure:"metrics"`
}

func (t Telemetry) Levels() map[string]any {
	levels := map[string]any{}
	for _, m := range t.Metrics {
		levels[m.Level.String()] = nil
	}
	return levels
}
