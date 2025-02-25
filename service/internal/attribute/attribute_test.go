// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package attribute_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/internal/telemetry/componentattribute"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/pipeline/xpipeline"
	"go.opentelemetry.io/collector/service/internal/attribute"
)

var (
	signals = []pipeline.Signal{
		pipeline.SignalTraces,
		pipeline.SignalMetrics,
		pipeline.SignalLogs,
		xpipeline.SignalProfiles,
	}

	cIDs = []component.ID{
		component.MustNewID("foo"),
		component.MustNewID("foo2"),
		component.MustNewID("bar"),
	}

	pIDs = []pipeline.ID{
		pipeline.MustNewID("traces"),
		pipeline.MustNewIDWithName("traces", "2"),
		pipeline.MustNewID("metrics"),
		pipeline.MustNewIDWithName("metrics", "2"),
		pipeline.MustNewID("logs"),
		pipeline.MustNewIDWithName("logs", "2"),
		pipeline.MustNewID("profiles"),
		pipeline.MustNewIDWithName("profiles", "2"),
	}
)

func TestReceiver(t *testing.T) {
	for _, sig := range signals {
		for _, id := range cIDs {
			r := attribute.Receiver(sig, id)
			componentKind, ok := r.Set().Value(componentattribute.ComponentKindKey)
			require.True(t, ok)
			require.Equal(t, component.KindReceiver.String(), componentKind.AsString())

			signal, ok := r.Set().Value(componentattribute.SignalKey)
			require.True(t, ok)
			require.Equal(t, sig.String(), signal.AsString())

			componentID, ok := r.Set().Value(componentattribute.ComponentIDKey)
			require.True(t, ok)
			require.Equal(t, id.String(), componentID.AsString())
		}
	}
}

func TestProcessor(t *testing.T) {
	for _, pID := range pIDs {
		for _, id := range cIDs {
			p := attribute.Processor(pID, id)
			componentKind, ok := p.Set().Value(componentattribute.ComponentKindKey)
			require.True(t, ok)
			require.Equal(t, component.KindProcessor.String(), componentKind.AsString())

			pipelineID, ok := p.Set().Value(componentattribute.PipelineIDKey)
			require.True(t, ok)
			require.Equal(t, pID.String(), pipelineID.AsString())

			componentID, ok := p.Set().Value(componentattribute.ComponentIDKey)
			require.True(t, ok)
			require.Equal(t, id.String(), componentID.AsString())
		}
	}
}

func TestExporter(t *testing.T) {
	for _, sig := range signals {
		for _, id := range cIDs {
			e := attribute.Exporter(sig, id)
			componentKind, ok := e.Set().Value(componentattribute.ComponentKindKey)
			require.True(t, ok)
			require.Equal(t, component.KindExporter.String(), componentKind.AsString())

			signal, ok := e.Set().Value(componentattribute.SignalKey)
			require.True(t, ok)
			require.Equal(t, sig.String(), signal.AsString())

			componentID, ok := e.Set().Value(componentattribute.ComponentIDKey)
			require.True(t, ok)
			require.Equal(t, id.String(), componentID.AsString())
		}
	}
}

func TestConnector(t *testing.T) {
	for _, exprSig := range signals {
		for _, rcvrSig := range signals {
			for _, id := range cIDs {
				c := attribute.Connector(exprSig, rcvrSig, id)
				componentKind, ok := c.Set().Value(componentattribute.ComponentKindKey)
				require.True(t, ok)
				require.Equal(t, component.KindConnector.String(), componentKind.AsString())

				signal, ok := c.Set().Value(componentattribute.SignalKey)
				require.True(t, ok)
				require.Equal(t, exprSig.String(), signal.AsString())

				signalOutput, ok := c.Set().Value(componentattribute.SignalOutputKey)
				require.True(t, ok)
				require.Equal(t, rcvrSig.String(), signalOutput.AsString())

				componentID, ok := c.Set().Value(componentattribute.ComponentIDKey)
				require.True(t, ok)
				require.Equal(t, id.String(), componentID.AsString())
			}
		}
	}
}

func TestExtension(t *testing.T) {
	e := attribute.Extension(component.MustNewID("foo"))
	componentKind, ok := e.Set().Value(componentattribute.ComponentKindKey)
	require.True(t, ok)
	require.Equal(t, component.KindExtension.String(), componentKind.AsString())
}

func TestSetEquality(t *testing.T) {
	// The sets are created independently but should be exactly equivalent.
	// We will ensure that corresponding elements are equal and that
	// non-corresponding elements are not equal.
	setI, setJ := createExampleSets(), createExampleSets()
	for i, ei := range setI {
		for j, ej := range setJ {
			if i == j {
				require.Equal(t, ei.ID(), ej.ID())
				si, sj := ei.Set(), ej.Set()
				require.True(t, si.Equals(sj))
			} else {
				require.NotEqual(t, ei.ID(), ej.ID())
				si, sj := ei.Set(), ej.Set()
				require.False(t, si.Equals(sj))
			}
		}
	}
}

func createExampleSets() []attribute.Attributes {
	sets := []attribute.Attributes{}

	// Receiver examples.
	for _, sig := range signals {
		for _, id := range cIDs {
			sets = append(sets, attribute.Receiver(sig, id))
		}
	}

	// Processor examples.
	for _, pID := range pIDs {
		for _, cID := range cIDs {
			sets = append(sets, attribute.Processor(pID, cID))
		}
	}

	// Exporter examples.
	for _, sig := range signals {
		for _, id := range cIDs {
			sets = append(sets, attribute.Exporter(sig, id))
		}
	}

	// Connector examples.
	for _, exprSig := range signals {
		for _, rcvrSig := range signals {
			for _, id := range cIDs {
				sets = append(sets, attribute.Connector(exprSig, rcvrSig, id))
			}
		}
	}

	// Capabilities examples.
	for _, pID := range pIDs {
		sets = append(sets, attribute.Capabilities(pID))
	}

	// Fanout examples.
	for _, pID := range pIDs {
		sets = append(sets, attribute.Fanout(pID))
	}

	return sets
}
