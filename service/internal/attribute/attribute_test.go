// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package attribute

import (
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/pipeline/xpipeline"
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
			r := Receiver(sig, id)
			componentKind, ok := r.Attributes().Value(componentKindKey)
			require.True(t, ok)
			require.Equal(t, component.KindReceiver.String(), componentKind.AsString())

			signal, ok := r.Attributes().Value(signalKey)
			require.True(t, ok)
			require.Equal(t, sig.String(), signal.AsString())

			componentID, ok := r.Attributes().Value(componentIDKey)
			require.True(t, ok)
			require.Equal(t, id.String(), componentID.AsString())
		}
	}
}

func TestProcessor(t *testing.T) {
	for _, pID := range pIDs {
		for _, id := range cIDs {
			p := Processor(pID, id)
			componentKind, ok := p.Attributes().Value(componentKindKey)
			require.True(t, ok)
			require.Equal(t, component.KindProcessor.String(), componentKind.AsString())

			pipelineID, ok := p.Attributes().Value(pipelineIDKey)
			require.True(t, ok)
			require.Equal(t, pID.String(), pipelineID.AsString())

			componentID, ok := p.Attributes().Value(componentIDKey)
			require.True(t, ok)
			require.Equal(t, id.String(), componentID.AsString())
		}
	}
}

func TestExporter(t *testing.T) {
	for _, sig := range signals {
		for _, id := range cIDs {
			e := Exporter(sig, id)
			componentKind, ok := e.Attributes().Value(componentKindKey)
			require.True(t, ok)
			require.Equal(t, component.KindExporter.String(), componentKind.AsString())

			signal, ok := e.Attributes().Value(signalKey)
			require.True(t, ok)
			require.Equal(t, sig.String(), signal.AsString())

			componentID, ok := e.Attributes().Value(componentIDKey)
			require.True(t, ok)
			require.Equal(t, id.String(), componentID.AsString())
		}
	}
}

func TestConnector(t *testing.T) {
	for _, exprSig := range signals {
		for _, rcvrSig := range signals {
			for _, id := range cIDs {
				c := Connector(exprSig, rcvrSig, id)
				componentKind, ok := c.Attributes().Value(componentKindKey)
				require.True(t, ok)
				require.Equal(t, component.KindConnector.String(), componentKind.AsString())

				signal, ok := c.Attributes().Value(signalKey)
				require.True(t, ok)
				require.Equal(t, exprSig.String(), signal.AsString())

				signalOutput, ok := c.Attributes().Value(signalOutputKey)
				require.True(t, ok)
				require.Equal(t, rcvrSig.String(), signalOutput.AsString())

				componentID, ok := c.Attributes().Value(componentIDKey)
				require.True(t, ok)
				require.Equal(t, id.String(), componentID.AsString())
			}
		}
	}
}

func TestExtension(t *testing.T) {
	e := Extension(component.MustNewID("foo"))
	componentKind, ok := e.Attributes().Value(componentKindKey)
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
				require.True(t, ei.Attributes().Equals(ej.Attributes()))
			} else {
				require.NotEqual(t, ei.ID(), ej.ID())
				require.False(t, ei.Attributes().Equals(ej.Attributes()))
			}
		}
	}
}

func createExampleSets() []*Attributes {
	sets := []*Attributes{}

	// Receiver examples.
	for _, sig := range signals {
		for _, id := range cIDs {
			sets = append(sets, Receiver(sig, id))
		}
	}

	// Processor examples.
	for _, pID := range pIDs {
		for _, cID := range cIDs {
			sets = append(sets, Processor(pID, cID))
		}
	}

	// Exporter examples.
	for _, sig := range signals {
		for _, id := range cIDs {
			sets = append(sets, Exporter(sig, id))
		}
	}

	// Connector examples.
	for _, exprSig := range signals {
		for _, rcvrSig := range signals {
			for _, id := range cIDs {
				sets = append(sets, Connector(exprSig, rcvrSig, id))
			}
		}
	}

	// Capabilities examples.
	for _, pID := range pIDs {
		sets = append(sets, Capabilities(pID))
	}

	// Fanout examples.
	for _, pID := range pIDs {
		sets = append(sets, Fanout(pID))
	}

	return sets
}
