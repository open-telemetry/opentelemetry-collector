// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package attribute

import (
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentprofiles"
	"go.opentelemetry.io/collector/pipeline"
)

var (
	signals = []pipeline.Signal{
		pipeline.SignalTraces,
		pipeline.SignalMetrics,
		pipeline.SignalLogs,
		componentprofiles.SignalProfiles,
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

func TestAttributes(t *testing.T) {
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
