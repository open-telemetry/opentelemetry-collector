// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package xexporterhelpertest provides test helpers for the xexporterhelper package,
// following the RFC component-interface guidelines for open/optional interfaces.
//
// Every public interface package SHOULD provide test helpers in a <package>test subpackage
// with NewNop() and NewErr() constructors. See docs/rfcs/component-interfaces.md.
//
// Experimental: This API is at the early stage of development and may change without backward
// compatibility until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
package xexporterhelpertest // import "go.opentelemetry.io/collector/exporter/exporterhelper/xexporterhelper/xexporterhelpertest"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterhelper/xexporterhelper"
	"go.opentelemetry.io/collector/extension"
)

// Compile-time assertions: baseMiddlewareExtension must satisfy both extension.Extension
// and xexporterhelper.RequestMiddleware.
var (
	_ extension.Extension               = (*baseMiddlewareExtension)(nil)
	_ xexporterhelper.RequestMiddleware = (*baseMiddlewareExtension)(nil)
)

// baseMiddlewareExtension embeds the function types for all relevant interfaces so that
// unset fields default to no-op behavior. Tests embed or override only the fields they care about.
type baseMiddlewareExtension struct {
	component.StartFunc
	component.ShutdownFunc
	xexporterhelper.WrapSenderFunc
}

// NewNop returns an extension.Extension that also implements xexporterhelper.RequestMiddleware.
// All methods have no-op behavior: Start and Shutdown succeed immediately, and WrapSender
// returns the next sender unchanged (because nil WrapSenderFunc is a no-op).
//
// Experimental: This API is at the early stage of development and may change without backward
// compatibility.
func NewNop() extension.Extension {
	return &baseMiddlewareExtension{}
}

// NewErr returns an extension.Extension that also implements xexporterhelper.RequestMiddleware.
// WrapSender always fails with err; Start and Shutdown remain no-ops to allow testing
// isolated error handling of WrapSender without lifecycle interference.
//
// Experimental: This API is at the early stage of development and may change without backward
// compatibility.
func NewErr(err error) extension.Extension {
	return &baseMiddlewareExtension{
		WrapSenderFunc: func(
			_ xexporterhelper.RequestMiddlewareSettings,
			_ xexporterhelper.Sender[xexporterhelper.Request],
		) (xexporterhelper.Sender[xexporterhelper.Request], error) {
			return nil, err
		},
	}
}
