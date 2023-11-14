// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import "context"

// QueueRequest defines a request coming through a queue.
type QueueRequest[T any] struct {
	Request                  T
	Context                  context.Context
	onProcessingFinishedFunc func()
}

func newQueueRequest[T any](ctx context.Context, req T) QueueRequest[T] {
	return QueueRequest[T]{
		Request: req,
		Context: ctx,
	}
}

// OnProcessingFinished calls the optional callback function to handle cleanup after all processing is finished
func (qr *QueueRequest[T]) OnProcessingFinished() {
	if qr.onProcessingFinishedFunc != nil {
		qr.onProcessingFinishedFunc()
	}
}
