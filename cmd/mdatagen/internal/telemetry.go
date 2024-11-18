// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/cmd/mdatagen/internal"

import (
	"go.opentelemetry.io/collector/config/configtelemetry"
)

type Telemetry struct {
	Level   configtelemetry.Level `mapstructure:"level"`
	Metrics map[MetricName]Metric `mapstructure:"metrics"`
}

func (t Telemetry) Levels() map[string]interface{} {
	levels := map[string]interface{}{}
	for _, m := range t.Metrics {
		levels[m.Level.String()] = nil
	}
	return levels
}
