// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatchprocessor // import "go.opentelemetry.io/collector/processor/queuebatchprocessor"

import (
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

// Config is queue/batch processor configuration shared with exporterhelper.
type Config = exporterhelper.QueueBatchConfig
