// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fanoutconsumer

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/internal/testdata"
)

func TestLogsNotMultiplexing(t *testing.T) {
	nop := consumertest.NewNop()
	lfc := NewLogs([]consumer.Logs{nop})
	assert.Same(t, nop, lfc)
}

func TestLogsMultiplexingNonMutating(t *testing.T) {
	p1 := new(consumertest.LogsSink)
	p2 := new(consumertest.LogsSink)
	p3 := new(consumertest.LogsSink)

	lfc := NewLogs([]consumer.Logs{p1, p2, p3})
	assert.False(t, lfc.Capabilities().MutatesData)
	ld := testdata.GenerateLogsOneLogRecord()

	for i := 0; i < 2; i++ {
		err := lfc.ConsumeLogs(context.Background(), ld)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	assert.True(t, ld == p1.AllLogs()[0])
	assert.True(t, ld == p1.AllLogs()[1])
	assert.EqualValues(t, ld, p1.AllLogs()[0])
	assert.EqualValues(t, ld, p1.AllLogs()[1])

	assert.True(t, ld == p2.AllLogs()[0])
	assert.True(t, ld == p2.AllLogs()[1])
	assert.EqualValues(t, ld, p2.AllLogs()[0])
	assert.EqualValues(t, ld, p2.AllLogs()[1])

	assert.True(t, ld == p3.AllLogs()[0])
	assert.True(t, ld == p3.AllLogs()[1])
	assert.EqualValues(t, ld, p3.AllLogs()[0])
	assert.EqualValues(t, ld, p3.AllLogs()[1])
}

func TestLogsMultiplexingMutating(t *testing.T) {
	p1 := &mutatingLogsSink{LogsSink: new(consumertest.LogsSink)}
	p2 := &mutatingLogsSink{LogsSink: new(consumertest.LogsSink)}
	p3 := &mutatingLogsSink{LogsSink: new(consumertest.LogsSink)}

	lfc := NewLogs([]consumer.Logs{p1, p2, p3})
	assert.False(t, lfc.Capabilities().MutatesData)
	ld := testdata.GenerateLogsOneLogRecord()

	for i := 0; i < 2; i++ {
		err := lfc.ConsumeLogs(context.Background(), ld)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	assert.True(t, ld != p1.AllLogs()[0])
	assert.True(t, ld != p1.AllLogs()[1])
	assert.EqualValues(t, ld, p1.AllLogs()[0])
	assert.EqualValues(t, ld, p1.AllLogs()[1])

	assert.True(t, ld != p2.AllLogs()[0])
	assert.True(t, ld != p2.AllLogs()[1])
	assert.EqualValues(t, ld, p2.AllLogs()[0])
	assert.EqualValues(t, ld, p2.AllLogs()[1])

	// For this consumer, will receive the initial data.
	assert.True(t, ld == p3.AllLogs()[0])
	assert.True(t, ld == p3.AllLogs()[1])
	assert.EqualValues(t, ld, p3.AllLogs()[0])
	assert.EqualValues(t, ld, p3.AllLogs()[1])
}

func TestLogsMultiplexingMixLastMutating(t *testing.T) {
	p1 := &mutatingLogsSink{LogsSink: new(consumertest.LogsSink)}
	p2 := new(consumertest.LogsSink)
	p3 := &mutatingLogsSink{LogsSink: new(consumertest.LogsSink)}

	lfc := NewLogs([]consumer.Logs{p1, p2, p3})
	assert.False(t, lfc.Capabilities().MutatesData)
	ld := testdata.GenerateLogsOneLogRecord()

	for i := 0; i < 2; i++ {
		err := lfc.ConsumeLogs(context.Background(), ld)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	assert.True(t, ld != p1.AllLogs()[0])
	assert.True(t, ld != p1.AllLogs()[1])
	assert.EqualValues(t, ld, p1.AllLogs()[0])
	assert.EqualValues(t, ld, p1.AllLogs()[1])

	// For this consumer, will receive the initial data.
	assert.True(t, ld == p2.AllLogs()[0])
	assert.True(t, ld == p2.AllLogs()[1])
	assert.EqualValues(t, ld, p2.AllLogs()[0])
	assert.EqualValues(t, ld, p2.AllLogs()[1])

	// For this consumer, will clone the initial data.
	assert.True(t, ld != p3.AllLogs()[0])
	assert.True(t, ld != p3.AllLogs()[1])
	assert.EqualValues(t, ld, p3.AllLogs()[0])
	assert.EqualValues(t, ld, p3.AllLogs()[1])
}

func TestLogsMultiplexingMixLastNonMutating(t *testing.T) {
	p1 := &mutatingLogsSink{LogsSink: new(consumertest.LogsSink)}
	p2 := &mutatingLogsSink{LogsSink: new(consumertest.LogsSink)}
	p3 := new(consumertest.LogsSink)

	lfc := NewLogs([]consumer.Logs{p1, p2, p3})
	assert.False(t, lfc.Capabilities().MutatesData)
	ld := testdata.GenerateLogsOneLogRecord()

	for i := 0; i < 2; i++ {
		err := lfc.ConsumeLogs(context.Background(), ld)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	assert.True(t, ld != p1.AllLogs()[0])
	assert.True(t, ld != p1.AllLogs()[1])
	assert.EqualValues(t, ld, p1.AllLogs()[0])
	assert.EqualValues(t, ld, p1.AllLogs()[1])

	assert.True(t, ld != p2.AllLogs()[0])
	assert.True(t, ld != p2.AllLogs()[1])
	assert.EqualValues(t, ld, p2.AllLogs()[0])
	assert.EqualValues(t, ld, p2.AllLogs()[1])

	// For this consumer, will receive the initial data.
	assert.True(t, ld == p3.AllLogs()[0])
	assert.True(t, ld == p3.AllLogs()[1])
	assert.EqualValues(t, ld, p3.AllLogs()[0])
	assert.EqualValues(t, ld, p3.AllLogs()[1])
}

func TestLogsWhenErrors(t *testing.T) {
	p1 := mutatingErr{Consumer: consumertest.NewErr(errors.New("my error"))}
	p2 := consumertest.NewErr(errors.New("my error"))
	p3 := new(consumertest.LogsSink)

	lfc := NewLogs([]consumer.Logs{p1, p2, p3})
	ld := testdata.GenerateLogsOneLogRecord()

	for i := 0; i < 2; i++ {
		assert.Error(t, lfc.ConsumeLogs(context.Background(), ld))
	}

	assert.True(t, ld == p3.AllLogs()[0])
	assert.True(t, ld == p3.AllLogs()[1])
	assert.EqualValues(t, ld, p3.AllLogs()[0])
	assert.EqualValues(t, ld, p3.AllLogs()[1])
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
