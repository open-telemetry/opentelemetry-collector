// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xexporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper/xexporterhelper"

import (
	"context"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requestmiddleware"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
)

// Sender is an alias for the internal sender interface, allowing extensions to reference it.
type Sender[T any] = sender.Sender[T]

// RequestMiddlewareSettings is an alias for the internal settings struct.
type RequestMiddlewareSettings = requestmiddleware.RequestMiddlewareSettings

// RequestMiddleware defines the interface for components that can intercept and
// modify the request lifecycle, such as Adaptive Concurrency Controllers.
type RequestMiddleware interface {
	// OnRequest is called before the request is sent.
	// It can block to implement concurrency control or return an error to cancel the request.
	OnRequest(ctx context.Context) (func(error), error)
}
