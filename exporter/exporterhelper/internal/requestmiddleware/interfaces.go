// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package requestmiddleware // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/requestmiddleware"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
	"go.opentelemetry.io/collector/pipeline"
)

// RequestMiddleware enables extensions to intercept or modify exporter requests.
// Implementations can wrap the downstream sender with additional logic (e.g., rate control, adaptive concurrency).

// RequestMiddlewareSettings provides context about the exporter request pipeline.
type RequestMiddlewareSettings struct {
	Signal    pipeline.Signal
	ID        component.ID
	Telemetry component.TelemetrySettings
}

// RequestMiddleware allows an extension to wrap an exporter's request sender.
type RequestMiddleware interface {
	// WrapSender allows the middleware to wrap the next sender in the chain.
	// It returns a generic sender that implements the specific request type.
	WrapSender(set RequestMiddlewareSettings, next sender.Sender[request.Request]) (sender.Sender[request.Request], error)
}
