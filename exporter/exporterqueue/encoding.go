// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterqueue // import "go.opentelemetry.io/collector/exporter/exporterqueue"
import (
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

// Deprecated: [v0.123.0] Use exporterhelper.ErrQueueIsFull
var ErrQueueIsFull = exporterhelper.ErrQueueIsFull

// Deprecated: [v0.123.0] Use exporterhelper.QueueBatchEncoding
// Duplicate definition with queuebatch.Encoding since aliasing generics is not supported by default.
type Encoding[T any] interface {
	// Marshal is a function that can marshal a request into bytes.
	Marshal(T) ([]byte, error)

	// Unmarshal is a function that can unmarshal bytes into a request.
	Unmarshal([]byte) (T, error)
}
