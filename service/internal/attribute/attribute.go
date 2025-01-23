// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package attribute // import "go.opentelemetry.io/collector/service/internal/attribute"

import (
	"hash/fnv"

	"go.opentelemetry.io/otel/attribute"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pipeline"
)

const (
	componentKindKey = "otelcol.component.kind"
	componentIDKey   = "otelcol.component.id"
	pipelineIDKey    = "otelcol.pipeline.id"
	signalKey        = "otelcol.signal"
	signalOutputKey  = "otelcol.signal.output"

	capabiltiesKind = "capabilities"
	fanoutKind      = "fanout"
)

type Attributes struct {
	set attribute.Set
	id  int64
}

func newAttributes(attrs ...attribute.KeyValue) *Attributes {
	h := fnv.New64a()
	for _, kv := range attrs {
		h.Write([]byte("(" + string(kv.Key) + "|" + kv.Value.AsString() + ")"))
	}
	return &Attributes{
		set: attribute.NewSet(attrs...),
		id:  int64(h.Sum64()), // #nosec G115
	}
}

func (a Attributes) Attributes() *attribute.Set {
	return &a.set
}

func (a Attributes) ID() int64 {
	return a.id
}

func Receiver(pipelineType pipeline.Signal, id component.ID) *Attributes {
	return newAttributes(
		attribute.String(componentKindKey, component.KindReceiver.String()),
		attribute.String(signalKey, pipelineType.String()),
		attribute.String(componentIDKey, id.String()),
	)
}

func Processor(pipelineID pipeline.ID, id component.ID) *Attributes {
	return newAttributes(
		attribute.String(componentKindKey, component.KindProcessor.String()),
		attribute.String(signalKey, pipelineID.Signal().String()),
		attribute.String(pipelineIDKey, pipelineID.String()),
		attribute.String(componentIDKey, id.String()),
	)
}

func Exporter(pipelineType pipeline.Signal, id component.ID) *Attributes {
	return newAttributes(
		attribute.String(componentKindKey, component.KindExporter.String()),
		attribute.String(signalKey, pipelineType.String()),
		attribute.String(componentIDKey, id.String()),
	)
}

func Connector(exprPipelineType, rcvrPipelineType pipeline.Signal, id component.ID) *Attributes {
	return newAttributes(
		attribute.String(componentKindKey, component.KindConnector.String()),
		attribute.String(signalKey, exprPipelineType.String()),
		attribute.String(signalOutputKey, rcvrPipelineType.String()),
		attribute.String(componentIDKey, id.String()),
	)
}

func Capabilities(pipelineID pipeline.ID) *Attributes {
	return newAttributes(
		attribute.String(componentKindKey, capabiltiesKind),
		attribute.String(pipelineIDKey, pipelineID.String()),
	)
}

func Fanout(pipelineID pipeline.ID) *Attributes {
	return newAttributes(
		attribute.String(componentKindKey, fanoutKind),
		attribute.String(pipelineIDKey, pipelineID.String()),
	)
}

func Extension(id component.ID) *Attributes {
	return newAttributes(
		attribute.String(componentKindKey, component.KindExtension.String()),
		attribute.String(componentIDKey, id.String()),
	)
}
