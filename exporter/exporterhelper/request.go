// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
)

// Request represents a single request that can be sent to an external endpoint.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type Request = request.Request

// RequestErrorHandler is an optional interface that can be implemented by Request to provide a way handle partial
// temporary failures. For example, if some items failed to process and can be retried, this interface allows to
// return a new Request that contains the items left to be sent. Otherwise, the original Request should be returned.
// If not implemented, the original Request will be returned assuming the error is applied to the whole Request.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type RequestErrorHandler = request.ErrorHandler

// RequestConverterFunc converts pdata telemetry into a user-defined Request.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type RequestConverterFunc[T any] func(context.Context, T) (Request, error)

// RequestConsumeFunc processes the request. After the function returns, the request is no longer accessible,
// and accessing it is considered undefined behavior.
type RequestConsumeFunc = sender.SendFunc[Request]

// RequestSizer is an interface that returns the size of the given request.
type RequestSizer = request.Sizer[Request]

// Deprecated: [v0.129.0] no need, always supported.
func NewRequestsSizer() RequestSizer {
	return request.RequestsSizer[Request]{}
}

type RequestSizerType = request.SizerType

var (
	RequestSizerTypeBytes    = request.SizerTypeBytes
	RequestSizerTypeItems    = request.SizerTypeItems
	RequestSizerTypeRequests = request.SizerTypeRequests
)
