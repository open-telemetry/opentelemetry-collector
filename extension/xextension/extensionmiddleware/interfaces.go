// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionmiddleware // import "go.opentelemetry.io/collector/extension/xextension/extensionmiddleware"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/pipeline"
)

// RequestMiddleware allows an extension to intercept and modify the execution of
// an exporter's request (e.g., for rate limiting, concurrency control, or circuit breaking).
type RequestMiddleware interface {
	// Handle is called by the exporter helper. It must call `next` to proceed with the export.
	Handle(ctx context.Context, next func(context.Context) error) error

	// Shutdown cleans up any resources used by the middleware.
	Shutdown(ctx context.Context) error
}

// RequestMiddlewareFactory is an interface for extensions that can create
// request middlewares for specific components.
type RequestMiddlewareFactory interface {
	extension.Extension

	// CreateRequestMiddleware creates a middleware for a specific exporter and signal.
	// This allows the extension to maintain separate limits per exporter/signal if desired.
	CreateRequestMiddleware(component.ID, pipeline.Signal) (RequestMiddleware, error)
}
