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
	metrics := MustNewID("metrics")
	logs := MustNewID("logs")
	receiver := MustNewID("receiver")

	id1 := NewInstanceID(receiver, KindReceiver, traces)
	id2 := InstanceIDWithPipelines(id1, metrics, logs)

	assert.Equal(t, receiver, id1.ComponentID())
	assert.Equal(t, KindReceiver, id1.Kind())
	assert.Equal(t, map[ID]struct{}{
		traces: {},
	}, id1.pipelineIDs)

	assert.Equal(t, receiver, id2.ComponentID())
	assert.Equal(t, KindReceiver, id2.Kind())
	assert.Equal(t, map[ID]struct{}{
		traces:  {},
		metrics: {},
		logs:    {},
	}, id2.pipelineIDs)
}
