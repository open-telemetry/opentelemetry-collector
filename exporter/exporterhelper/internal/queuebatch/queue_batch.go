// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"

import (
	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/exporter/exporterqueue"
)

type Settings[K any] struct {
	Encoding exporterqueue.Encoding[K]
	Sizers   map[exporterbatcher.SizerType]Sizer[K]
}
