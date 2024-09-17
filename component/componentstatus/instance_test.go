// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componentstatus

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pipeline"
)

func TestInstanceID(t *testing.T) {
	traces := component.MustNewID("traces")
	tracesA := pipeline.MustNewIDWithName("traces", "a")
	tracesB := pipeline.MustNewIDWithName("traces", "b")
	tracesC := pipeline.MustNewIDWithName("traces", "c")

	idTracesA := NewInstanceIDWithPipelineIDs(traces, component.KindReceiver, tracesA)
	idTracesAll := NewInstanceIDWithPipelineIDs(traces, component.KindReceiver, tracesA, tracesB, tracesC)
	assert.NotEqual(t, idTracesA, idTracesAll)

	assertHasPipelines := func(t *testing.T, instanceID *InstanceID, expectedPipelineIDs []pipeline.ID) {
		var pipelineIDs []pipeline.ID
		instanceID.AllPipelineIDsWithPipelineIDs(func(id pipeline.ID) bool {
			pipelineIDs = append(pipelineIDs, id)
			return true
		})
		assert.Equal(t, expectedPipelineIDs, pipelineIDs)
	}

	for _, tc := range []struct {
		name        string
		id1         *InstanceID
		id2         *InstanceID
		pipelineIDs []pipeline.ID
	}{
		{
			name:        "equal instances",
			id1:         idTracesA,
			id2:         NewInstanceIDWithPipelineIDs(traces, component.KindReceiver, tracesA),
			pipelineIDs: []pipeline.ID{tracesA},
		},
		{
			name:        "equal instances - out of order",
			id1:         idTracesAll,
			id2:         NewInstanceIDWithPipelineIDs(traces, component.KindReceiver, tracesC, tracesB, tracesA),
			pipelineIDs: []pipeline.ID{tracesA, tracesB, tracesC},
		},
		{
			name:        "with pipelines",
			id1:         idTracesAll,
			id2:         idTracesA.WithPipelineIDs(tracesB, tracesC),
			pipelineIDs: []pipeline.ID{tracesA, tracesB, tracesC},
		},
		{
			name:        "with pipelines - out of order",
			id1:         idTracesAll,
			id2:         idTracesA.WithPipelineIDs(tracesC, tracesB),
			pipelineIDs: []pipeline.ID{tracesA, tracesB, tracesC},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.id1, tc.id2)
			assertHasPipelines(t, tc.id1, tc.pipelineIDs)
			assertHasPipelines(t, tc.id2, tc.pipelineIDs)
		})
	}
}

func TestAllPipelineIDsWithPipelineIDs(t *testing.T) {
	instanceID := NewInstanceIDWithPipelineIDs(
		component.MustNewID("traces"),
		component.KindReceiver,
		pipeline.MustNewIDWithName("traces", "a"),
		pipeline.MustNewIDWithName("traces", "b"),
		pipeline.MustNewIDWithName("traces", "c"),
	)

	count := 0
	instanceID.AllPipelineIDsWithPipelineIDs(func(pipeline.ID) bool {
		count++
		return true
	})
	assert.Equal(t, 3, count)

	count = 0
	instanceID.AllPipelineIDsWithPipelineIDs(func(pipeline.ID) bool {
		count++
		return false
	})
	assert.Equal(t, 1, count)

}
