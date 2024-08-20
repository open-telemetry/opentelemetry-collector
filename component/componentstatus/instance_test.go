// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componentstatus

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/component"
)

func TestInstanceID(t *testing.T) {
	traces := component.MustNewID("traces")
	tracesA := component.MustNewIDWithName("traces", "a")
	tracesB := component.MustNewIDWithName("traces", "b")
	tracesC := component.MustNewIDWithName("traces", "c")

	idTracesA := NewInstanceID(traces, component.KindReceiver, tracesA)
	idTracesAll := NewInstanceID(traces, component.KindReceiver, tracesA, tracesB, tracesC)
	assert.NotEqual(t, idTracesA, idTracesAll)

	assertHasPipelines := func(t *testing.T, instanceID *InstanceID, expectedPipelineIDs []component.ID) {
		var pipelineIDs []component.ID
		instanceID.AllPipelineIDs(func(id component.ID) bool {
			pipelineIDs = append(pipelineIDs, id)
			return true
		})
		assert.Equal(t, expectedPipelineIDs, pipelineIDs)
	}

	for _, tc := range []struct {
		name        string
		id1         *InstanceID
		id2         *InstanceID
		pipelineIDs []component.ID
	}{
		{
			name:        "equal instances",
			id1:         idTracesA,
			id2:         NewInstanceID(traces, component.KindReceiver, tracesA),
			pipelineIDs: []component.ID{tracesA},
		},
		{
			name:        "equal instances - out of order",
			id1:         idTracesAll,
			id2:         NewInstanceID(traces, component.KindReceiver, tracesC, tracesB, tracesA),
			pipelineIDs: []component.ID{tracesA, tracesB, tracesC},
		},
		{
			name:        "with pipelines",
			id1:         idTracesAll,
			id2:         idTracesA.WithPipelines(tracesB, tracesC),
			pipelineIDs: []component.ID{tracesA, tracesB, tracesC},
		},
		{
			name:        "with pipelines - out of order",
			id1:         idTracesAll,
			id2:         idTracesA.WithPipelines(tracesC, tracesB),
			pipelineIDs: []component.ID{tracesA, tracesB, tracesC},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.id1, tc.id2)
			assertHasPipelines(t, tc.id1, tc.pipelineIDs)
			assertHasPipelines(t, tc.id2, tc.pipelineIDs)
		})
	}
}

func TestAllPipelineIDs(t *testing.T) {
	instanceID := NewInstanceID(
		component.MustNewID("traces"),
		component.KindReceiver,
		component.MustNewIDWithName("traces", "a"),
		component.MustNewIDWithName("traces", "b"),
		component.MustNewIDWithName("traces", "c"),
	)

	count := 0
	instanceID.AllPipelineIDs(func(component.ID) bool {
		count++
		return true
	})
	assert.Equal(t, 3, count)

	count = 0
	instanceID.AllPipelineIDs(func(component.ID) bool {
		count++
		return false
	})
	assert.Equal(t, 1, count)

}
