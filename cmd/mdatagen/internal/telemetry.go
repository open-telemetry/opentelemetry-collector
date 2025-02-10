// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/cmd/mdatagen/internal"

type Telemetry struct {
	Metrics map[MetricName]Metric `mapstructure:"metrics"`
}
