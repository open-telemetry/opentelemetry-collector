// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package request // import "go.opentelemetry.io/collector/exporter/exporterhelper/request"

import (
	"context"
)

// Request represents a single request that can be sent to an external endpoint.
// This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type Request interface {
	// Export exports the request to an external endpoint.
	Export(ctx context.Context) error
}

// ItemsCounter is an optional interface that can be implemented by Request to provide a number of items
// in the request. This is a recommended interface to implement for exporters. It is required for batching and queueing
// based on number of items. Also, it's used for reporting number of items in collector's logs, metrics and traces.
// If not implemented, collector's logs, metrics and traces will report 0 items.
// This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type ItemsCounter interface {
	// ItemsCount returns a number of basic items in the request where item is the smallest piece of data that can be
	// sent. For example, for OTLP exporter, this value represents the number of spans,
	// metric data points or log records.
	ItemsCount() int
}

// ErrorHandler is an optional interface that can be implemented by Request to provide a way handle partial
// temporary failures. For example, if some items failed to process and can be retried, this interface allows to
// return a new Request that contains the items left to be sent. Otherwise, the original Request should be returned.
// If not implemented, the original Request will be returned assuming the error is applied to the whole Request.
// This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type ErrorHandler interface {
	// OnError returns a new Request may contain the items left to be sent if some items failed to process and can be retried.
	// Otherwise, it should return the original Request.
	OnError(error) Request
}

// Marshaler is a function that can marshal a Request into bytes.
// This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type Marshaler func(req Request) ([]byte, error)

// Unmarshaler is a function that can unmarshal bytes into a Request.
// This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type Unmarshaler func(data []byte) (Request, error)
