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
	"go.opentelemetry.io/collector/pdata/plog"
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
	ld := testdata.GenerateLogs(1)

	for i := 0; i < 2; i++ {
		err := lfc.ConsumeLogs(context.Background(), ld)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	assert.True(t, ld.ResourceLogs() == p1.AllLogs()[0].ResourceLogs())
	assert.True(t, ld.ResourceLogs() == p1.AllLogs()[1].ResourceLogs())

	assert.True(t, ld.ResourceLogs() == p2.AllLogs()[0].ResourceLogs())
	assert.True(t, ld.ResourceLogs() == p2.AllLogs()[1].ResourceLogs())

	assert.True(t, ld.ResourceLogs() == p3.AllLogs()[0].ResourceLogs())
	assert.True(t, ld.ResourceLogs() == p3.AllLogs()[1].ResourceLogs())
}

func TestLogsMultiplexingMutating(t *testing.T) {
	p1 := &mutatingLogsSink{LogsSink: new(consumertest.LogsSink)}
	p2 := &mutatingLogsSink{LogsSink: new(consumertest.LogsSink)}
	p3 := &mutatingLogsSink{LogsSink: new(consumertest.LogsSink)}

	lfc := NewLogs([]consumer.Logs{p1, p2, p3})
	ld := testdata.GenerateLogs(1)

	for i := 0; i < 2; i++ {
		err := lfc.ConsumeLogs(context.Background(), ld)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	// All the consumers receive shared logs that are cloned by calling MutableResourceLogs.

	assert.True(t, ld.ResourceLogs() != p1.AllLogs()[0].ResourceLogs())
	assert.True(t, ld.ResourceLogs() != p1.AllLogs()[1].ResourceLogs())
	assert.EqualValues(t, ld, p1.AllLogs()[0])
	assert.EqualValues(t, ld, p1.AllLogs()[1])

	assert.True(t, ld.ResourceLogs() != p2.AllLogs()[0].ResourceLogs())
	assert.True(t, ld.ResourceLogs() != p2.AllLogs()[1].ResourceLogs())
	assert.EqualValues(t, ld, p2.AllLogs()[0])
	assert.EqualValues(t, ld, p2.AllLogs()[1])

	assert.True(t, ld.ResourceLogs() != p3.AllLogs()[0].ResourceLogs())
	assert.True(t, ld.ResourceLogs() != p3.AllLogs()[1].ResourceLogs())
	assert.EqualValues(t, ld, p3.AllLogs()[0])
	assert.EqualValues(t, ld, p3.AllLogs()[1])
}

func TestLogsMultiplexingMixLastMutating(t *testing.T) {
	p1 := &mutatingLogsSink{LogsSink: new(consumertest.LogsSink)}
	p2 := new(consumertest.LogsSink)
	p3 := &mutatingLogsSink{LogsSink: new(consumertest.LogsSink)}

	lfc := NewLogs([]consumer.Logs{p1, p2, p3})
	ld := testdata.GenerateLogs(1)

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
	assert.True(t, ld.ResourceLogs() == p2.AllLogs()[0].ResourceLogs())
	assert.True(t, ld.ResourceLogs() == p2.AllLogs()[1].ResourceLogs())

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
	ld := testdata.GenerateLogs(1)

	for i := 0; i < 2; i++ {
		err := lfc.ConsumeLogs(context.Background(), ld)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	assert.True(t, ld.ResourceLogs() != p1.AllLogs()[0].ResourceLogs())
	assert.True(t, ld.ResourceLogs() != p1.AllLogs()[1].ResourceLogs())
	assert.EqualValues(t, ld, p1.AllLogs()[0])
	assert.EqualValues(t, ld, p1.AllLogs()[1])

	assert.True(t, ld.ResourceLogs() != p2.AllLogs()[0].ResourceLogs())
	assert.True(t, ld.ResourceLogs() != p2.AllLogs()[1].ResourceLogs())
	assert.EqualValues(t, ld, p2.AllLogs()[0])
	assert.EqualValues(t, ld, p2.AllLogs()[1])

	// For this consumer, will receive the initial data.
	assert.True(t, ld.ResourceLogs() == p3.AllLogs()[0].ResourceLogs())
	assert.True(t, ld.ResourceLogs() == p3.AllLogs()[1].ResourceLogs())
}

func TestLogsWhenErrors(t *testing.T) {
	p1 := mutatingErrLogs{errors.New("my error")}
	p2 := consumertest.NewErr(errors.New("my error"))
	p3 := new(consumertest.LogsSink)

	lfc := NewLogs([]consumer.Logs{p1, p2, p3})
	ld := testdata.GenerateLogs(1)

	for i := 0; i < 2; i++ {
		assert.Error(t, lfc.ConsumeLogs(context.Background(), ld))
	}

	assert.True(t, ld.ResourceLogs() == p3.AllLogs()[0].ResourceLogs())
	assert.True(t, ld.ResourceLogs() == p3.AllLogs()[1].ResourceLogs())
}

type mutatingLogsSink struct {
	*consumertest.LogsSink
}

func (mts *mutatingLogsSink) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	_ = ld.MutableResourceLogs()
	return mts.LogsSink.ConsumeLogs(ctx, ld)
}

type mutatingErrLogs struct {
	err error
}

func (mts mutatingErrLogs) ConsumeLogs(_ context.Context, ld plog.Logs) error {
	_ = ld.MutableResourceLogs()
	return mts.err
}
