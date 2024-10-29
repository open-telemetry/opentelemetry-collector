// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/internal"

import (
	"context"

	"go.opentelemetry.io/collector/exporter/exporterbatcher"
)

// Request represents a single request that can be sent to an external endpoint.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type Request interface {
	// Export exports the request to an external endpoint.
	Export(ctx context.Context) error
	// ItemsCount returns a number of basic items in the request where item is the smallest piece of data that can be
	// sent. For example, for OTLP exporter, this value represents the number of spans,
	// metric data points or log records.
	ItemsCount() int
	// Merge() is a function that merges this request with another one into a single request.
	// Do not mutate the requests passed to the function if error can be returned after mutation or if the exporter is
	// marked as not mutable.
	// Experimental: This API is at the early stage of development and may change without backward compatibility
	// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
	Merge(context.Context, Request) (Request, error)
	// MergeSplit() is a function that merge and/or splits this request with another one into multiple requests based on the
	// configured limit provided in MaxSizeConfig.
	// All the returned requests MUST have a number of items that does not exceed the maximum number of items.
	// Size of the last returned request MUST be less or equal than the size of any other returned request.
	// The original request MUST not be mutated if error is returned after mutation or if the exporter is
	// marked as not mutable. The length of the returned slice MUST not be 0.
	// Experimental: This API is at the early stage of development and may change without backward compatibility
	// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
	MergeSplit(context.Context, exporterbatcher.MaxSizeConfig, Request) ([]Request, error)

	BytesSize() int
}

// RequestErrorHandler is an optional interface that can be implemented by Request to provide a way handle partial
// temporary failures. For example, if some items failed to process and can be retried, this interface allows to
// return a new Request that contains the items left to be sent. Otherwise, the original Request should be returned.
// If not implemented, the original Request will be returned assuming the error is applied to the whole Request.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type RequestErrorHandler interface {
	Request
	// OnError returns a new Request may contain the items left to be sent if some items failed to process and can be retried.
	// Otherwise, it should return the original Request.
	OnError(error) Request
}
