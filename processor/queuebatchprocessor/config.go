// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatchprocessor // import "go.opentelemetry.io/collector/processor/queuebatchprocessor"

import (
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

// Config is the configuration for the queue/batch processor. It is the same
// combined queue/batch configuration used by exporterhelper, so the processor
// supports every queueing and batching mode the exporter helper supports.
type Config = exporterhelper.QueueBatchConfig
