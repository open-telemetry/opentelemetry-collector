// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatchprocessor // import "go.opentelemetry.io/collector/processor/queuebatchprocessor"

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/embedded"

	"go.opentelemetry.io/collector/processor/queuebatchprocessor/internal/metadata"
)

// The embedded exporterhelper emits its enqueue and export spans under an
// "exporter/" name namespace and under the exporterhelper instrumentation
// scope. Because the exporterhelper is an implementation detail of this
// processor, those spans read as exporter spans.
//
// renamingTracerProvider reframes them: it reports the spans under this
// processor's instrumentation scope and rewrites the "exporter/" span-name
// prefix to "processor/". Everything else (attributes, links, parent, kind,
// timing) is passed through unchanged. In a running collector the service
// already injects otelcol.component.{kind,id} and otelcol.pipeline.id as scope
// attributes (see service/internal/componentattribute), which identify these
// spans as belonging to this processor.
//
// This is coupled to exporterhelper's "exporter/" span-name prefix; if that
// changes upstream, update renameSpan.
const (
	exporterSpanPrefix  = "exporter/"
	processorSpanPrefix = "processor/"
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
	return t.delegate.Start(ctx, renameSpan(name), opts...)
}

func renameSpan(name string) string {
	if after, ok := strings.CutPrefix(name, exporterSpanPrefix); ok {
		return processorSpanPrefix + after
	}
	return name
}
