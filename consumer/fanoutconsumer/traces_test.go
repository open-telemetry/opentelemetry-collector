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

func TestTracesNotMultiplexing(t *testing.T) {
	nop := consumertest.NewNop()
	tfc := NewTraces([]consumer.Traces{nop})
	assert.Same(t, nop, tfc)
}

func TestTracesMultiplexingNonMutating(t *testing.T) {
	p1 := new(consumertest.TracesSink)
	p2 := new(consumertest.TracesSink)
	p3 := new(consumertest.TracesSink)

	tfc := NewTraces([]consumer.Traces{p1, p2, p3})
	assert.False(t, tfc.Capabilities().MutatesData)
	td := testdata.GenerateTraces(1)

	for i := 0; i < 2; i++ {
		err := tfc.ConsumeTraces(context.Background(), td)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	assert.True(t, td == p1.AllTraces()[0])
	assert.True(t, td == p1.AllTraces()[1])
	assert.EqualValues(t, td, p1.AllTraces()[0])
	assert.EqualValues(t, td, p1.AllTraces()[1])

	assert.True(t, td == p2.AllTraces()[0])
	assert.True(t, td == p2.AllTraces()[1])
	assert.EqualValues(t, td, p2.AllTraces()[0])
	assert.EqualValues(t, td, p2.AllTraces()[1])

	assert.True(t, td == p3.AllTraces()[0])
	assert.True(t, td == p3.AllTraces()[1])
	assert.EqualValues(t, td, p3.AllTraces()[0])
	assert.EqualValues(t, td, p3.AllTraces()[1])
}

func TestTracesMultiplexingMutating(t *testing.T) {
	p1 := &mutatingTracesSink{TracesSink: new(consumertest.TracesSink)}
	p2 := &mutatingTracesSink{TracesSink: new(consumertest.TracesSink)}
	p3 := &mutatingTracesSink{TracesSink: new(consumertest.TracesSink)}

	tfc := NewTraces([]consumer.Traces{p1, p2, p3})
	assert.False(t, tfc.Capabilities().MutatesData)
	td := testdata.GenerateTraces(1)

	for i := 0; i < 2; i++ {
		err := tfc.ConsumeTraces(context.Background(), td)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	assert.True(t, td != p1.AllTraces()[0])
	assert.True(t, td != p1.AllTraces()[1])
	assert.EqualValues(t, td, p1.AllTraces()[0])
	assert.EqualValues(t, td, p1.AllTraces()[1])

	assert.True(t, td != p2.AllTraces()[0])
	assert.True(t, td != p2.AllTraces()[1])
	assert.EqualValues(t, td, p2.AllTraces()[0])
	assert.EqualValues(t, td, p2.AllTraces()[1])

	// For this consumer, will receive the initial data.
	assert.True(t, td == p3.AllTraces()[0])
	assert.True(t, td == p3.AllTraces()[1])
	assert.EqualValues(t, td, p3.AllTraces()[0])
	assert.EqualValues(t, td, p3.AllTraces()[1])
}

func TestTracesMultiplexingMixLastMutating(t *testing.T) {
	p1 := &mutatingTracesSink{TracesSink: new(consumertest.TracesSink)}
	p2 := new(consumertest.TracesSink)
	p3 := &mutatingTracesSink{TracesSink: new(consumertest.TracesSink)}

	tfc := NewTraces([]consumer.Traces{p1, p2, p3})
	assert.False(t, tfc.Capabilities().MutatesData)
	td := testdata.GenerateTraces(1)

	for i := 0; i < 2; i++ {
		err := tfc.ConsumeTraces(context.Background(), td)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	assert.True(t, td != p1.AllTraces()[0])
	assert.True(t, td != p1.AllTraces()[1])
	assert.EqualValues(t, td, p1.AllTraces()[0])
	assert.EqualValues(t, td, p1.AllTraces()[1])

	// For this consumer, will receive the initial data.
	assert.True(t, td == p2.AllTraces()[0])
	assert.True(t, td == p2.AllTraces()[1])
	assert.EqualValues(t, td, p2.AllTraces()[0])
	assert.EqualValues(t, td, p2.AllTraces()[1])

	// For this consumer, will clone the initial data.
	assert.True(t, td != p3.AllTraces()[0])
	assert.True(t, td != p3.AllTraces()[1])
	assert.EqualValues(t, td, p3.AllTraces()[0])
	assert.EqualValues(t, td, p3.AllTraces()[1])
}

func TestTracesMultiplexingMixLastNonMutating(t *testing.T) {
	p1 := &mutatingTracesSink{TracesSink: new(consumertest.TracesSink)}
	p2 := &mutatingTracesSink{TracesSink: new(consumertest.TracesSink)}
	p3 := new(consumertest.TracesSink)

	tfc := NewTraces([]consumer.Traces{p1, p2, p3})
	assert.False(t, tfc.Capabilities().MutatesData)
	td := testdata.GenerateTraces(1)

	for i := 0; i < 2; i++ {
		err := tfc.ConsumeTraces(context.Background(), td)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	assert.True(t, td != p1.AllTraces()[0])
	assert.True(t, td != p1.AllTraces()[1])
	assert.EqualValues(t, td, p1.AllTraces()[0])
	assert.EqualValues(t, td, p1.AllTraces()[1])

	assert.True(t, td != p2.AllTraces()[0])
	assert.True(t, td != p2.AllTraces()[1])
	assert.EqualValues(t, td, p2.AllTraces()[0])
	assert.EqualValues(t, td, p2.AllTraces()[1])

	// For this consumer, will receive the initial data.
	assert.True(t, td == p3.AllTraces()[0])
	assert.True(t, td == p3.AllTraces()[1])
	assert.EqualValues(t, td, p3.AllTraces()[0])
	assert.EqualValues(t, td, p3.AllTraces()[1])
}

func TestTracesWhenErrors(t *testing.T) {
	p1 := mutatingErr{Consumer: consumertest.NewErr(errors.New("my error"))}
	p2 := consumertest.NewErr(errors.New("my error"))
	p3 := new(consumertest.TracesSink)

	tfc := NewTraces([]consumer.Traces{p1, p2, p3})
	td := testdata.GenerateTraces(1)

	for i := 0; i < 2; i++ {
		assert.Error(t, tfc.ConsumeTraces(context.Background(), td))
	}

	assert.True(t, td == p3.AllTraces()[0])
	assert.True(t, td == p3.AllTraces()[1])
	assert.EqualValues(t, td, p3.AllTraces()[0])
	assert.EqualValues(t, td, p3.AllTraces()[1])
}

type mutatingTracesSink struct {
	*consumertest.TracesSink
}

func (mts *mutatingTracesSink) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}
