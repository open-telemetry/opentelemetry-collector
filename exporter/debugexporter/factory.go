// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package debugexporter // import "go.opentelemetry.io/collector/exporter/debugexporter"

import (
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/loggingexporter"
)

const (
	// The value of "type" key in configuration.
	typeStr = "debug"
)

// NewFactory creates a factory for Logging exporter
func NewFactory() exporter.Factory {
	return loggingexporter.NewFactoryWithName(typeStr)
}
