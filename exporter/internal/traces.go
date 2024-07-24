// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/internal"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
)

// Traces is an exporter that can consume traces.
type Traces interface {
	component.Component
	consumer.Traces
}
