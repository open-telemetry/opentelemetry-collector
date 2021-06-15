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
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/extension/storage"
)

func createStorageExtension(_ string) storage.Extension {
	// After having storage moved to core, we could leverage storagetest.NewTestExtension(nil, path)
	return newMockStorageExtension()
}

func createTestQueue(extension storage.Extension, capacity int) *persistentQueue {
	logger, _ := zap.NewDevelopment()

	client, err := extension.GetClient(context.Background(), component.KindReceiver, config.ComponentID{}, "")
	if err != nil {
		panic(err)
	}

	wq := newPersistentQueue(context.Background(), "foo", capacity, logger, client, newTraceRequestUnmarshalerFunc(nopTracePusher()))
	wq.storage.(*persistentContiguousStorage).mu.Lock()
	wq.storage.(*persistentContiguousStorage).retryDelay = 10 * time.Millisecond
	wq.storage.(*persistentContiguousStorage).mu.Unlock()
	return wq
}

func createTestPersistentStorage(extension storage.Extension) persistentStorage {
	logger, _ := zap.NewDevelopment()

	client, err := extension.GetClient(context.Background(), component.KindReceiver, config.ComponentID{}, "")
	if err != nil {
		panic(err)
	}

	return newPersistentContiguousStorage(context.Background(), "foo", logger, client, newTraceRequestUnmarshalerFunc(nopTracePusher()))
}

func createTemporaryDirectory() string {
	directory, err := ioutil.TempDir("", "persistent-test")
	if err != nil {
		panic(err)
	}
	return directory
}

func TestPersistentQueue_RepeatPutCloseReadClose(t *testing.T) {
	path := createTemporaryDirectory()
	defer os.RemoveAll(path)

	traces := newTraces(5, 10)
	req := newTracesRequest(context.Background(), traces, nopTracePusher())

	for i := 0; i < 10; i++ {
		ext := createStorageExtension(path)
		ps := createTestPersistentStorage(ext)
		require.Equal(t, 0, ps.size())
		err := ps.put(req)
		require.NoError(t, err)
		err = ext.Shutdown(context.Background())
		require.NoError(t, err)

		// TODO: when replacing mock with real storage, this could actually be uncommented
		// ext = createStorageExtension(path)
		// ps = createTestPersistentStorage(ext)

		require.Equal(t, 1, ps.size())
		readReq := <-ps.get()
		require.Equal(t, 0, ps.size())
		require.Equal(t, req.(*tracesRequest).td, readReq.(*tracesRequest).td)
		err = ext.Shutdown(context.Background())
		require.NoError(t, err)
	}

	// No more items
	ext := createStorageExtension(path)
	wq := createTestQueue(ext, 5000)
	require.Equal(t, 0, wq.Size())
	ext.Shutdown(context.Background())
}

func TestPersistentQueue_Capacity(t *testing.T) {
	path := createTemporaryDirectory()
	defer os.RemoveAll(path)

	ext := createStorageExtension(path)
	defer ext.Shutdown(context.Background())

	wq := createTestQueue(ext, 5)
	require.Equal(t, 0, wq.Size())

	traces := newTraces(1, 10)
	req := newTracesRequest(context.Background(), traces, nopTracePusher())

	for i := 0; i < 10; i++ {
		result := wq.Produce(req)
		if i < 5 {
			require.True(t, result)
		} else {
			require.False(t, result)
		}
	}
	require.Equal(t, 5, wq.Size())
}

func TestPersistentQueue_ConsumersProducers(t *testing.T) {
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

			traces := newTraces(1, 10)
			req := newTracesRequest(context.Background(), traces, nopTracePusher())

			ext := createStorageExtension(path)
			tq := createTestQueue(ext, 5000)

			defer os.RemoveAll(path)
			defer tq.Stop()
			defer ext.Shutdown(context.Background())

			numMessagesConsumed := int32(0)
			tq.StartConsumers(c.numConsumers, func(item interface{}) {
				if item != nil {
					atomic.AddInt32(&numMessagesConsumed, 1)
				}
			})

			for i := 0; i < c.numMessagesProduced; i++ {
				tq.Produce(req)
			}

			require.Eventually(t, func() bool {
				return c.numMessagesProduced == int(atomic.LoadInt32(&numMessagesConsumed))
			}, 500*time.Millisecond, 10*time.Millisecond)
		})
	}
}

func BenchmarkPersistentQueue_1Trace10Spans(b *testing.B) {
	cases := []struct {
		numTraces        int
		numSpansPerTrace int
	}{
		{
			numTraces:        1,
			numSpansPerTrace: 1,
		},
		{
			numTraces:        1,
			numSpansPerTrace: 10,
		},
		{
			numTraces:        10,
			numSpansPerTrace: 10,
		},
	}

	for _, c := range cases {
		b.Run(fmt.Sprintf("#traces: %d #spansPerTrace: %d", c.numTraces, c.numSpansPerTrace), func(bb *testing.B) {
			path := createTemporaryDirectory()
			defer os.RemoveAll(path)
			ext := createStorageExtension(path)
			ps := createTestPersistentStorage(ext)

			traces := newTraces(c.numTraces, c.numSpansPerTrace)
			req := newTracesRequest(context.Background(), traces, nopTracePusher())

			bb.ResetTimer()

			for i := 0; i < bb.N; i++ {
				err := ps.put(req)
				require.NoError(bb, err)
			}

			for i := 0; i < bb.N; i++ {
				req := ps.get()
				require.NotNil(bb, req)
			}
			ext.Shutdown(context.Background())
		})
	}
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

type mockStorageExtension struct{}

func (m mockStorageExtension) Start(_ context.Context, _ component.Host) error {
	return nil
}

func (m mockStorageExtension) Shutdown(_ context.Context) error {
	return nil
}

func (m mockStorageExtension) GetClient(ctx context.Context, kind component.Kind, id config.ComponentID, s string) (storage.Client, error) {
	return newMockStorageClient(), nil
}

func newMockStorageExtension() storage.Extension {
	return &mockStorageExtension{}
}

func newMockStorageClient() storage.Client {
	return &mockStorageClient{
		st: map[string][]byte{},
	}
}

type mockStorageClient struct {
	st map[string][]byte
}

func (m mockStorageClient) Get(_ context.Context, s string) ([]byte, error) {
	val, found := m.st[s]
	if !found {
		return []byte{}, errors.New("key not found")
	}

	return val, nil
}

func (m mockStorageClient) Set(_ context.Context, s string, bytes []byte) error {
	m.st[s] = bytes
	return nil
}

func (m mockStorageClient) Delete(_ context.Context, s string) error {
	delete(m.st, s)
	return nil
}

func (m mockStorageClient) Close(_ context.Context) error {
	return nil
}
