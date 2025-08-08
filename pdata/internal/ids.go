// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/pdata/internal"

import (
	"go.opentelemetry.io/collector/pdata/internal/data"
)

func SizeProtoOrigTraceID(id *data.TraceID) int {
	return id.Size()
}

func SizeProtoOrigSpanID(id *data.SpanID) int {
	return id.Size()
}

func SizeProtoOrigProfileID(id *data.ProfileID) int {
	return id.Size()
}
