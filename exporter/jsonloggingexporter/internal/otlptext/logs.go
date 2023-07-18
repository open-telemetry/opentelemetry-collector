// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlptext // import "go.opentelemetry.io/collector/exporter/jsonloggingexporter/internal/otlptext"

import (
	"go.opentelemetry.io/collector/pdata/plog"
)

// NewJsonLogsMarshaler returns a plog.Marshaler to encode to OTLP JSON text bytes.
func NewJsonLogsMarshaler() plog.Marshaler {
	return &plog.JSONMarshaler{}
}
