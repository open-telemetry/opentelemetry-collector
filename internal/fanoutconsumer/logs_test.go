// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package fanoutconsumer

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/testdata"
)

func TestLogsNotMultiplexing(t *testing.T) {
	nop := consumertest.NewNop()
	lfc := NewLogs([]consumer.Logs{nop})
	assert.Same(t, nop, lfc)
}

func TestLogsNotMultiplexingMutating(t *testing.T) {
	p := &mutatingLogsSink{LogsSink: new(consumertest.LogsSink)}
	lfc := NewLogs([]consumer.Logs{p})
	assert.True(t, lfc.Capabilities().MutatesData)
}

func TestLogsMultiplexingNonMutating(t *testing.T) {
	p1 := new(consumertest.LogsSink)
	p2 := new(consumertest.LogsSink)
	p3 := new(consumertest.LogsSink)

	lfc := NewLogs([]consumer.Logs{p1, p2, p3})
	assert.False(t, lfc.Capabilities().MutatesData)
	ld := testdata.GenerateLogs(1)

	for range 2 {
		err := lfc.ConsumeLogs(context.Background(), ld)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	assert.Equal(t, ld, p1.AllLogs()[0])
	assert.Equal(t, ld, p1.AllLogs()[1])
	assert.Equal(t, ld, p1.AllLogs()[0])
	assert.Equal(t, ld, p1.AllLogs()[1])

	assert.Equal(t, ld, p2.AllLogs()[0])
	assert.Equal(t, ld, p2.AllLogs()[1])
	assert.Equal(t, ld, p2.AllLogs()[0])
	assert.Equal(t, ld, p2.AllLogs()[1])

	assert.Equal(t, ld, p3.AllLogs()[0])
	assert.Equal(t, ld, p3.AllLogs()[1])
	assert.Equal(t, ld, p3.AllLogs()[0])
	assert.Equal(t, ld, p3.AllLogs()[1])

	// The data should be marked as read only.
	assert.True(t, ld.IsReadOnly())
}

func TestLogsMultiplexingMutating(t *testing.T) {
	p1 := &mutatingLogsSink{LogsSink: new(consumertest.LogsSink)}
	p2 := &mutatingLogsSink{LogsSink: new(consumertest.LogsSink)}
	p3 := &mutatingLogsSink{LogsSink: new(consumertest.LogsSink)}

	lfc := NewLogs([]consumer.Logs{p1, p2, p3})
	assert.True(t, lfc.Capabilities().MutatesData)
	ld := testdata.GenerateLogs(1)

	for range 2 {
		err := lfc.ConsumeLogs(context.Background(), ld)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	assert.NotSame(t, &ld, &p1.AllLogs()[0])
	assert.NotSame(t, &ld, &p1.AllLogs()[1])
	assert.Equal(t, ld, p1.AllLogs()[0])
	assert.Equal(t, ld, p1.AllLogs()[1])

	assert.NotSame(t, &ld, &p2.AllLogs()[0])
	assert.NotSame(t, &ld, &p2.AllLogs()[1])
	assert.Equal(t, ld, p2.AllLogs()[0])
	assert.Equal(t, ld, p2.AllLogs()[1])

	// For this consumer, will receive the initial data.
	assert.Equal(t, ld, p3.AllLogs()[0])
	assert.Equal(t, ld, p3.AllLogs()[1])
	assert.Equal(t, ld, p3.AllLogs()[0])
	assert.Equal(t, ld, p3.AllLogs()[1])

	// The data should not be marked as read only.
	assert.False(t, ld.IsReadOnly())
}

func TestReadOnlyLogsMultiplexingMutating(t *testing.T) {
	p1 := &mutatingLogsSink{LogsSink: new(consumertest.LogsSink)}
	p2 := &mutatingLogsSink{LogsSink: new(consumertest.LogsSink)}
	p3 := &mutatingLogsSink{LogsSink: new(consumertest.LogsSink)}

	lfc := NewLogs([]consumer.Logs{p1, p2, p3})
	assert.True(t, lfc.Capabilities().MutatesData)
	ldOrig := testdata.GenerateLogs(1)
	ld := testdata.GenerateLogs(1)
	ld.MarkReadOnly()

	for range 2 {
		err := lfc.ConsumeLogs(context.Background(), ld)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	// All consumers should receive the cloned data.

	assert.NotEqual(t, ld, p1.AllLogs()[0])
	assert.NotEqual(t, ld, p1.AllLogs()[1])
	assert.Equal(t, ldOrig, p1.AllLogs()[0])
	assert.Equal(t, ldOrig, p1.AllLogs()[1])

	assert.NotEqual(t, ld, p2.AllLogs()[0])
	assert.NotEqual(t, ld, p2.AllLogs()[1])
	assert.Equal(t, ldOrig, p2.AllLogs()[0])
	assert.Equal(t, ldOrig, p2.AllLogs()[1])

	assert.NotEqual(t, ld, p3.AllLogs()[0])
	assert.NotEqual(t, ld, p3.AllLogs()[1])
	assert.Equal(t, ldOrig, p3.AllLogs()[0])
	assert.Equal(t, ldOrig, p3.AllLogs()[1])
}

func TestLogsMultiplexingMixLastMutating(t *testing.T) {
	p1 := &mutatingLogsSink{LogsSink: new(consumertest.LogsSink)}
	p2 := new(consumertest.LogsSink)
	p3 := &mutatingLogsSink{LogsSink: new(consumertest.LogsSink)}

	lfc := NewLogs([]consumer.Logs{p1, p2, p3})
	assert.False(t, lfc.Capabilities().MutatesData)
	ld := testdata.GenerateLogs(1)

	for range 2 {
		err := lfc.ConsumeLogs(context.Background(), ld)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	assert.NotSame(t, &ld, &p1.AllLogs()[0])
	assert.NotSame(t, &ld, &p1.AllLogs()[1])
	assert.Equal(t, ld, p1.AllLogs()[0])
	assert.Equal(t, ld, p1.AllLogs()[1])

	// For this consumer, will receive the initial data.
	assert.Equal(t, ld, p2.AllLogs()[0])
	assert.Equal(t, ld, p2.AllLogs()[1])
	assert.Equal(t, ld, p2.AllLogs()[0])
	assert.Equal(t, ld, p2.AllLogs()[1])

	// For this consumer, will clone the initial data.
	assert.NotSame(t, &ld, &p3.AllLogs()[0])
	assert.NotSame(t, &ld, &p3.AllLogs()[1])
	assert.Equal(t, ld, p3.AllLogs()[0])
	assert.Equal(t, ld, p3.AllLogs()[1])

	// The data should not be marked as read only.
	assert.False(t, ld.IsReadOnly())
}

func TestLogsMultiplexingMixLastNonMutating(t *testing.T) {
	p1 := &mutatingLogsSink{LogsSink: new(consumertest.LogsSink)}
	p2 := &mutatingLogsSink{LogsSink: new(consumertest.LogsSink)}
	p3 := new(consumertest.LogsSink)

	lfc := NewLogs([]consumer.Logs{p1, p2, p3})
	assert.False(t, lfc.Capabilities().MutatesData)
	ld := testdata.GenerateLogs(1)

	for range 2 {
		err := lfc.ConsumeLogs(context.Background(), ld)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	assert.NotSame(t, &ld, &p1.AllLogs()[0])
	assert.NotSame(t, &ld, &p1.AllLogs()[1])
	assert.Equal(t, ld, p1.AllLogs()[0])
	assert.Equal(t, ld, p1.AllLogs()[1])

	assert.NotSame(t, &ld, &p2.AllLogs()[0])
	assert.NotSame(t, &ld, &p2.AllLogs()[1])
	assert.Equal(t, ld, p2.AllLogs()[0])
	assert.Equal(t, ld, p2.AllLogs()[1])

	// For this consumer, will receive the initial data.
	assert.Equal(t, ld, p3.AllLogs()[0])
	assert.Equal(t, ld, p3.AllLogs()[1])
	assert.Equal(t, ld, p3.AllLogs()[0])
	assert.Equal(t, ld, p3.AllLogs()[1])

	// The data should not be marked as read only.
	assert.False(t, ld.IsReadOnly())
}

func TestLogsWhenErrors(t *testing.T) {
	p1 := mutatingErr{Consumer: consumertest.NewErr(errors.New("my error"))}
	p2 := consumertest.NewErr(errors.New("my error"))
	p3 := new(consumertest.LogsSink)

	lfc := NewLogs([]consumer.Logs{p1, p2, p3})
	ld := testdata.GenerateLogs(1)

	for range 2 {
		require.Error(t, lfc.ConsumeLogs(context.Background(), ld))
	}

	assert.Equal(t, ld, p3.AllLogs()[0])
	assert.Equal(t, ld, p3.AllLogs()[1])
	assert.Equal(t, ld, p3.AllLogs()[0])
	assert.Equal(t, ld, p3.AllLogs()[1])
}

type mutatingLogsSink struct {
	*consumertest.LogsSink
}

func (mts *mutatingLogsSink) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

type mutatingErr struct {
	consumertest.Consumer
}

func (mts mutatingErr) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}
