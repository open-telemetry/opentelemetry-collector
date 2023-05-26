// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterbatcher

import (
	"context"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
)

// Batch is a batch of items that can be queued before sending in a single request.
type Batch interface {
	// Size returns the size of the batch in bytes.
	Size() int

	// Count returns the number of items in the batch.
	Count() int

	// Append appends the given batch to the current batch and returns the new batch.
	Append(Batch) Batch // Rename to Merge?

	// Marshal serializes the batch into bytes.
	Marshal() ([]byte, error)

	// Export sends the batch out.
	Export(context.Context) error
}

type batchRequest struct {
	internal.BaseRequest
	Batch
}

func (req *batchRequest) OnError(_ error) internal.Request {
	return req
}

var _ internal.Request = (*batchRequest)(nil)
