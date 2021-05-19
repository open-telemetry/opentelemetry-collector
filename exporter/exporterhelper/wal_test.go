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

package exporterhelper

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/consumer/pdata"
)

func createTestQueue(path string) *WALQueue {
	logger, _ := zap.NewDevelopment()

	wq, err := newWALQueue(logger, path, "foo", 100*time.Millisecond, newTraceRequestUnmarshalerFunc(nopTracePusher()))
	if err != nil {
		panic(err)
	}
	return wq
}

func createTemporaryDirectory() string {
	directory, err := ioutil.TempDir("", "wal-test")
	if err != nil {
		panic(err)
	}
	return directory
}

func TestWal_RepeatPutCloseReadClose(t *testing.T) {
	path := createTemporaryDirectory()
	defer os.RemoveAll(path)

	traces := newTraces(1, 10)
	req := newTracesRequest(context.Background(), traces, nopTracePusher())

	for i := 0; i < 10; i++ {
		wq := createTestQueue(path)
		require.Equal(t, 0, wq.Size())
		err := wq.put(req)
		require.NoError(t, err)
		err = wq.close()
		require.NoError(t, err)

		wq = createTestQueue(path)
		require.Equal(t, 1, wq.Size())
		readReq, err := wq.get()
		require.NoError(t, err)
		require.Equal(t, 0, wq.Size())
		require.Equal(t, req.(*tracesRequest).td, readReq.(*tracesRequest).td)
		err = wq.close()
		require.NoError(t, err)
	}

	// No more items
	wq := createTestQueue(path)
	require.Equal(t, 0, wq.Size())
	wq.close()
}

func TestWal_ConsumersProducers(t *testing.T) {
	cases := []struct {
		numMessagesProduced int
		numConsumers        int
	}{
		{
			numMessagesProduced: 1,
			numConsumers:        1,
		},
		{
			numMessagesProduced: 100,
			numConsumers:        1,
		},
		{
			numMessagesProduced: 100,
			numConsumers:        3,
		},
		{
			numMessagesProduced: 1,
			numConsumers:        100,
		},
		{
			numMessagesProduced: 100,
			numConsumers:        100,
		},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("#messages: %d #consumers: %d", c.numMessagesProduced, c.numConsumers), func(t *testing.T) {
			path := createTemporaryDirectory()
			defer os.RemoveAll(path)

			traces := newTraces(1, 10)
			req := newTracesRequest(context.Background(), traces, nopTracePusher())

			wq := createTestQueue(path)

			numMessagesConsumed := 0
			wq.StartConsumers(c.numConsumers, func(item interface{}) {
				numMessagesConsumed++
			})

			for i := 0; i < c.numMessagesProduced; i++ {
				wq.Produce(req)
			}

			// TODO: proper handling/wait with timeout rather than this
			time.Sleep(500 * time.Millisecond)
			wq.close()
			require.Equal(t, c.numMessagesProduced, numMessagesConsumed)
		})
	}
}

func BenchmarkWal_1Trace10Spans(b *testing.B) {
	path := createTemporaryDirectory()
	wq := createTestQueue(path)
	defer os.RemoveAll(path)

	traces := newTraces(1, 10)
	req := newTracesRequest(context.Background(), traces, nopTracePusher())

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := wq.put(req)
		require.NoError(b, err)
	}

	for i := 0; i < b.N; i++ {
		req, err := wq.get()
		require.NoError(b, err)
		require.NotNil(b, req)
	}

	wq.close()
}

func newTraces(numTraces int, numSpans int) pdata.Traces {
	traces := pdata.NewTraces()
	batch := traces.ResourceSpans().AppendEmpty()
	batch.Resource().Attributes().InsertString("resource-attr", "some-resource")
	batch.Resource().Attributes().InsertInt("num-traces", int64(numTraces))
	batch.Resource().Attributes().InsertInt("num-spans", int64(numSpans))

	for i := 0; i < numTraces; i++ {
		traceID := pdata.NewTraceID([16]byte{1, 2, 3, byte(i)})
		ils := batch.InstrumentationLibrarySpans().AppendEmpty()
		for j := 0; j < numSpans; j++ {
			span := ils.Spans().AppendEmpty()
			span.SetTraceID(traceID)
			span.SetSpanID(pdata.NewSpanID([8]byte{1, 2, 3, byte(j)}))
			span.SetName("should-not-be-changed")
			span.Attributes().InsertInt("int-attribute", int64(j))
			span.Attributes().InsertString("str-attribute-1", "foobar")
			span.Attributes().InsertString("str-attribute-2", "fdslafjasdk12312312jkl")
			span.Attributes().InsertString("str-attribute-3", "AbcDefGeKKjkfdsafasdfsdasdf")
			span.Attributes().InsertString("str-attribute-4", "xxxxxx")
			span.Attributes().InsertString("str-attribute-5", "abcdef")
		}
	}

	return traces
}
