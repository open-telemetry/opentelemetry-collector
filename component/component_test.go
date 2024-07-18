// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package component

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKindString(t *testing.T) {
	assert.EqualValues(t, "", Kind(0).String())
	assert.EqualValues(t, "Receiver", KindReceiver.String())
	assert.EqualValues(t, "Processor", KindProcessor.String())
	assert.EqualValues(t, "Exporter", KindExporter.String())
	assert.EqualValues(t, "Extension", KindExtension.String())
	assert.EqualValues(t, "Connector", KindConnector.String())
	assert.EqualValues(t, "", Kind(100).String())
}

func TestStabilityLevelString(t *testing.T) {
	assert.EqualValues(t, "Undefined", StabilityLevelUndefined.String())
	assert.EqualValues(t, "Unmaintained", StabilityLevelUnmaintained.String())
	assert.EqualValues(t, "Deprecated", StabilityLevelDeprecated.String())
	assert.EqualValues(t, "Development", StabilityLevelDevelopment.String())
	assert.EqualValues(t, "Alpha", StabilityLevelAlpha.String())
	assert.EqualValues(t, "Beta", StabilityLevelBeta.String())
	assert.EqualValues(t, "Stable", StabilityLevelStable.String())
	assert.EqualValues(t, "", StabilityLevel(100).String())
}

func TestInstanceID(t *testing.T) {
	traces := MustNewID("traces")
	tracesA := MustNewIDWithName("traces", "a")
	tracesB := MustNewIDWithName("traces", "b")
	tracesC := MustNewIDWithName("traces", "c")

	idTracesA := NewInstanceID(traces, KindReceiver, tracesA)
	idTracesAll := NewInstanceID(traces, KindReceiver, tracesA, tracesB, tracesC)
	assert.NotEqual(t, idTracesA, idTracesAll)

	assertHasPipelines := func(t *testing.T, instanceID *InstanceID, expectedPipelineIDs []ID) {
		var pipelineIDs []ID
		instanceID.EachPipelineID(func(id ID) bool {
			pipelineIDs = append(pipelineIDs, id)
			return true
		})
		assert.Equal(t, expectedPipelineIDs, pipelineIDs)
	}

	for _, tc := range []struct {
		name        string
		id1         *InstanceID
		id2         *InstanceID
		pipelineIDs []ID
	}{
		{
			name:        "equal instances",
			id1:         idTracesA,
			id2:         NewInstanceID(traces, KindReceiver, tracesA),
			pipelineIDs: []ID{tracesA},
		},
		{
			name:        "equal instances - out of order",
			id1:         idTracesAll,
			id2:         NewInstanceID(traces, KindReceiver, tracesC, tracesB, tracesA),
			pipelineIDs: []ID{tracesA, tracesB, tracesC},
		},
		{
			name:        "with pipelines",
			id1:         idTracesAll,
			id2:         idTracesA.WithPipelines(tracesB, tracesC),
			pipelineIDs: []ID{tracesA, tracesB, tracesC},
		},
		{
			name:        "with pipelines - out of order",
			id1:         idTracesAll,
			id2:         idTracesA.WithPipelines(tracesC, tracesB),
			pipelineIDs: []ID{tracesA, tracesB, tracesC},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.id1, tc.id2)
			assertHasPipelines(t, tc.id1, tc.pipelineIDs)
			assertHasPipelines(t, tc.id2, tc.pipelineIDs)
		})
	}
}

func TestInstanceIDEachPipelineID(t *testing.T) {
	instanceID := NewInstanceID(
		MustNewID("traces"),
		KindReceiver,
		MustNewIDWithName("traces", "a"),
		MustNewIDWithName("traces", "b"),
		MustNewIDWithName("traces", "c"),
	)

	count := 0
	instanceID.EachPipelineID(func(id ID) bool {
		count++
		return true
	})
	assert.Equal(t, 3, count)

	count = 0
	instanceID.EachPipelineID(func(id ID) bool {
		count++
		return false
	})
	assert.Equal(t, 1, count)

}
