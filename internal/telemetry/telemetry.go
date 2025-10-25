// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/internal/telemetry"

import (
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/featuregate"
)

var NewPipelineTelemetryGate = featuregate.GlobalRegistry().MustRegister(
	"telemetry.newPipelineTelemetry",
	featuregate.StageAlpha,
	featuregate.WithRegisterFromVersion("v0.123.0"),
	featuregate.WithRegisterReferenceURL("https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/rfcs/component-universal-telemetry.md"),
	featuregate.WithRegisterDescription("Injects component-identifying scope attributes in internal Collector metrics"),
)

type injectorCore interface {
	DropInjectedAttributes(droppedAttrs ...string) zapcore.Core
}

type injectorTracerProvider interface {
	DropInjectedAttributes(droppedAttrs ...string) trace.TracerProvider
}

type injectorMeterProvider interface {
	DropInjectedAttributes(droppedAttrs ...string) metric.MeterProvider
}

func DropInjectedAttributes(ts component.TelemetrySettings, attrs ...string) component.TelemetrySettings {
	ts.Logger = ts.Logger.WithOptions(zap.WrapCore(func(c zapcore.Core) zapcore.Core {
		if ic, ok := c.(injectorCore); ok {
			return ic.DropInjectedAttributes(attrs...)
		}
		return c
	}))
	if itp, ok := ts.TracerProvider.(injectorTracerProvider); ok {
		ts.TracerProvider = itp.DropInjectedAttributes(attrs...)
	}
	if imp, ok := ts.MeterProvider.(injectorMeterProvider); ok {
		ts.MeterProvider = imp.DropInjectedAttributes(attrs...)
	}
	return ts
}
