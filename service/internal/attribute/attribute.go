// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package attribute // import "go.opentelemetry.io/collector/service/internal/attribute"

import (
	"hash/fnv"

	"go.opentelemetry.io/otel/attribute"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/internal/telemetry/componentattribute"
	"go.opentelemetry.io/collector/pipeline"
)

const (
	capabiltiesKind = "capabilities"
	fanoutKind      = "fanout"
)

type Attributes struct {
	set attribute.Set
	id  int64
}

func newAttributes(attrs ...attribute.KeyValue) Attributes {
	h := fnv.New64a()
	for _, kv := range attrs {
		h.Write([]byte("(" + string(kv.Key) + "|" + kv.Value.AsString() + ")"))
	}
	return Attributes{
		set: attribute.NewSet(attrs...),
		id:  int64(h.Sum64()), // #nosec G115
	}
}

func (a Attributes) Set() *attribute.Set {
	return &a.set
}

func (a Attributes) ID() int64 {
	return a.id
}

func Receiver(pipelineType pipeline.Signal, id component.ID) Attributes {
	return newAttributes(
		attribute.String(componentattribute.ComponentKindKey, component.KindReceiver.String()),
		attribute.String(componentattribute.SignalKey, pipelineType.String()),
		attribute.String(componentattribute.ComponentIDKey, id.String()),
	)
}

func Processor(pipelineID pipeline.ID, id component.ID) Attributes {
	return newAttributes(
		attribute.String(componentattribute.ComponentKindKey, component.KindProcessor.String()),
		attribute.String(componentattribute.SignalKey, pipelineID.Signal().String()),
		attribute.String(componentattribute.PipelineIDKey, pipelineID.String()),
		attribute.String(componentattribute.ComponentIDKey, id.String()),
	)
}

func Exporter(pipelineType pipeline.Signal, id component.ID) Attributes {
	return newAttributes(
		attribute.String(componentattribute.ComponentKindKey, component.KindExporter.String()),
		attribute.String(componentattribute.SignalKey, pipelineType.String()),
		attribute.String(componentattribute.ComponentIDKey, id.String()),
	)
}

func Connector(exprPipelineType, rcvrPipelineType pipeline.Signal, id component.ID) Attributes {
	return newAttributes(
		attribute.String(componentattribute.ComponentKindKey, component.KindConnector.String()),
		attribute.String(componentattribute.SignalKey, exprPipelineType.String()),
		attribute.String(componentattribute.SignalOutputKey, rcvrPipelineType.String()),
		attribute.String(componentattribute.ComponentIDKey, id.String()),
	)
}

func Extension(id component.ID) Attributes {
	return newAttributes(
		attribute.String(componentattribute.ComponentKindKey, component.KindExtension.String()),
		attribute.String(componentattribute.ComponentIDKey, id.String()),
	)
}

func Capabilities(pipelineID pipeline.ID) Attributes {
	return newAttributes(
		attribute.String(componentattribute.ComponentKindKey, capabiltiesKind),
		attribute.String(componentattribute.PipelineIDKey, pipelineID.String()),
	)
}

func Fanout(pipelineID pipeline.ID) Attributes {
	return newAttributes(
		attribute.String(componentattribute.ComponentKindKey, fanoutKind),
		attribute.String(componentattribute.PipelineIDKey, pipelineID.String()),
	)
}
