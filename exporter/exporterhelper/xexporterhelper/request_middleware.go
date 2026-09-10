// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xexporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper/xexporterhelper"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requestmiddleware"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
	"go.opentelemetry.io/collector/pipeline"
)

// Sender is an alias for the internal sender interface, allowing extensions to reference it.
type Sender[T any] = sender.Sender[T]

// RequestMiddlewareSettings provides context about the exporter request pipeline.
type RequestMiddlewareSettings = requestmiddleware.RequestMiddlewareSettings

// RequestMiddleware is an open interface for extensions that intercept or modify exporter requests.
// Extensions implement this interface to wrap the sender with additional logic such as adaptive concurrency control,
// circuit breaking, rate limiting, or request execution policy injection.
//
// This is an open interface for capability detection. Extensions implement it and exporterhelper discovers
// it via type assertion. Future capabilities should use companion interfaces rather than adding methods to this interface.
//
// Experimental: This API is at the early stage of development and may change without backward compatibility.
type RequestMiddleware interface {
	// WrapSender wraps next with additional request execution logic and returns
	// the new outermost sender. The returned sender's Start and Shutdown methods
	// are called directly by the exporterhelper infrastructure; they MUST NOT
	// cascade those calls into next, because next's lifecycle is managed
	// independently. A wrapper that calls next.Start or next.Shutdown will
	// cause double-start or double-shutdown when multiple middlewares are stacked.
	//
	// Experimental: This API is at the early stage of development and may change without backward compatibility.
	WrapSender(settings RequestMiddlewareSettings, next Sender[Request]) (Sender[Request], error)
}

// WrapSenderFunc is the function type corresponding to RequestMiddleware.WrapSender.
// It allows for nil-safe no-op behavior and can be embedded to automatically satisfy the interface.
//
// Experimental: This API is at the early stage of development and may change without backward compatibility.
type WrapSenderFunc func(settings RequestMiddlewareSettings, next Sender[Request]) (Sender[Request], error)

// Compile-time assertion that WrapSenderFunc satisfies RequestMiddleware.
var _ RequestMiddleware = (*WrapSenderFunc)(nil)

// WrapSender calls f with the provided arguments. If f is nil, it returns next unchanged.
// This provides nil-safe no-op behavior as required by the component interface guidelines.
func (f WrapSenderFunc) WrapSender(settings RequestMiddlewareSettings, next Sender[Request]) (Sender[Request], error) {
	if f == nil {
		return next, nil
	}
	return f(settings, next)
}

// NewRequestMiddlewareSettings creates a new RequestMiddlewareSettings from the given parameters.
func NewRequestMiddlewareSettings(id component.ID, signal pipeline.Signal, telemetry component.TelemetrySettings) RequestMiddlewareSettings {
	return RequestMiddlewareSettings{
		ID:        id,
		Signal:    signal,
		Telemetry: telemetry,
	}
}
