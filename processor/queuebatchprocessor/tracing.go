// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatchprocessor // import "go.opentelemetry.io/collector/processor/queuebatchprocessor"

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/embedded"

	"go.opentelemetry.io/collector/processor/queuebatchprocessor/internal/metadata"
)

// The embedded exporterhelper emits its spans under an "exporter/" name
// namespace, with "exporter" and "data_type" attributes, under the
// exporterhelper instrumentation scope. Because the exporterhelper is an
// implementation detail of this processor, those spans read as exporter spans.
//
// renamingTracerProvider rewrites them to read as processor spans: it reports
// them under this processor's instrumentation scope and maps the span names and
// attributes into the "processor/" namespace (matching this component's
// metrics), while preserving parents, links, span kind, and timing.
//
// This is coupled to exporterhelper's current span names and attribute keys; if
// those change upstream, the mappings below must be updated.
const (
	exporterSpanPrefix  = "exporter/"
	processorSpanPrefix = "processor/"
	exporterAttrKey     = "exporter"
	dataTypeAttrKey     = "data_type"
)

type renamingTracerProvider struct {
	embedded.TracerProvider
	delegate trace.TracerProvider
}

func newRenamingTracerProvider(delegate trace.TracerProvider) trace.TracerProvider {
	return renamingTracerProvider{delegate: delegate}
}

func (p renamingTracerProvider) Tracer(_ string, opts ...trace.TracerOption) trace.Tracer {
	// Report the spans under this processor's scope rather than exporterhelper's.
	return renamingTracer{delegate: p.delegate.Tracer(metadata.ScopeName, opts...)}
}

type renamingTracer struct {
	embedded.Tracer
	delegate trace.Tracer
}

func (t renamingTracer) Start(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	cfg := trace.NewSpanStartConfig(opts...)
	newOpts := []trace.SpanStartOption{
		trace.WithAttributes(remapSpanAttributes(cfg.Attributes())...),
		trace.WithLinks(cfg.Links()...),
		trace.WithSpanKind(cfg.SpanKind()),
	}
	if cfg.NewRoot() {
		newOpts = append(newOpts, trace.WithNewRoot())
	}
	if ts := cfg.Timestamp(); !ts.IsZero() {
		newOpts = append(newOpts, trace.WithTimestamp(ts))
	}
	return t.delegate.Start(ctx, renameSpan(name), newOpts...)
}

func renameSpan(name string) string {
	if after, ok := strings.CutPrefix(name, exporterSpanPrefix); ok {
		return processorSpanPrefix + after
	}
	return name
}

// remapSpanAttributes rewrites the exporterhelper span attribute keys to the
// processor conventions used by this component's metrics (processor and
// otel.signal), keeping the values unchanged.
func remapSpanAttributes(attrs []attribute.KeyValue) []attribute.KeyValue {
	out := make([]attribute.KeyValue, len(attrs))
	for i, a := range attrs {
		switch a.Key {
		case exporterAttrKey:
			a.Key = processorKey
		case dataTypeAttrKey:
			a.Key = signalKey
		}
		out[i] = a
	}
	return out
}
