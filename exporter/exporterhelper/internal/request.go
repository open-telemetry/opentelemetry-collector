// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import "context"

// QueueRequest defines a request coming through a queue.
type QueueRequest struct {
	Request                  any
	Context                  context.Context
	onProcessingFinishedFunc func()
}

func newQueueRequest(ctx context.Context, req any) QueueRequest {
	return QueueRequest{
		Request: req,
		Context: ctx,
	}
}

// OnProcessingFinished calls the optional callback function to handle cleanup after all processing is finished
func (qr *QueueRequest) OnProcessingFinished() {
	if qr.onProcessingFinishedFunc != nil {
		qr.onProcessingFinishedFunc()
	}
}

type QueueRequestMarshaler func(req any) ([]byte, error)

type QueueRequestUnmarshaler func(data []byte) (any, error)
