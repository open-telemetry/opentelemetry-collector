// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xexporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper/xexporterhelper"

import (
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requestmiddleware"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
)

// Sender is an alias for the internal sender interface, allowing extensions to reference it.
type Sender[T any] = sender.Sender[T]

// RequestMiddlewareSettings is an alias for the internal settings struct.
type RequestMiddlewareSettings = requestmiddleware.RequestMiddlewareSettings

// RequestMiddleware is an alias for the internal interface, allowing external extensions to implement it.
type RequestMiddleware = requestmiddleware.RequestMiddleware
