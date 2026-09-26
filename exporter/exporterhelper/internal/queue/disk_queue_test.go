// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/hosttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/extension/diskqueueextension"
	"go.opentelemetry.io/collector/extension/extensiontest"
	xqueue "go.opentelemetry.io/collector/extension/xextension/queue"
	"go.opentelemetry.io/collector/pipeline"
)

// newDiskAccessExtensionForTest creates a real diskaccess extension backed by a
// temp dir, so the disk queue can be exercised end-to-end without a fake.
func newDiskAccessExtensionForTest(tb testing.TB, dir string) component.Component {
	f := diskqueueextension.NewFactory()
	cfg := f.CreateDefaultConfig().(*diskqueueextension.Config)
	cfg.DataPath = dir
	cfg.SyncTimeout = 10 * time.Millisecond
	settings := extensiontest.NewNopSettings(component.MustNewType(diskqueueextension.TypeStr))
	ext, err := f.Create(context.Background(), settings, cfg)
	require.NoError(tb, err)
	return ext
}

func newSettingsWithDiskStorage(sizerType request.SizerType, capacity int64) (Settings[intRequest], component.ID) {
	set := newSettings(sizerType, capacity)
	storageID := component.MustNewID(diskqueueextension.TypeStr)
	set.StorageID = &storageID
	return set, storageID
}

// createTestDiskQueue starts a diskQueue backed by a real diskaccess
// extension. Callers are responsible for calling Shutdown exactly once,
// same as the memory/persistent queue test helpers.
func createTestDiskQueue(tb testing.TB, sizerType request.SizerType, capacity int64) *diskQueue[intRequest] {
	set, storageID := newSettingsWithDiskStorage(sizerType, capacity)
	dq := newDiskQueue[intRequest](set).(*diskQueue[intRequest])
	ext := newDiskAccessExtensionForTest(tb, tb.TempDir())
	host := hosttest.NewHost(map[component.ID]component.Component{storageID: ext})
	require.NoError(tb, dq.Start(context.Background(), host))
	return dq
}

func TestDiskQueue(t *testing.T) {
	dq := createTestDiskQueue(t, request.SizerTypeItems, 7)
	assert.EqualValues(t, 7, dq.Capacity())
	assert.EqualValues(t, 0, dq.Size())

	require.NoError(t, dq.Offer(context.Background(), 1))
	assert.EqualValues(t, 1, dq.Size())

	require.NoError(t, dq.Offer(context.Background(), 3))
	assert.EqualValues(t, 4, dq.Size())

	// should not be able to send to the full queue
	require.ErrorIs(t, dq.Offer(context.Background(), 4), ErrQueueIsFull)
	assert.EqualValues(t, 4, dq.Size())

	assert.True(t, consume(dq, func(_ context.Context, el intRequest) error {
		assert.EqualValues(t, 1, el)
		return nil
	}))
	assert.Eventually(t, func() bool { return dq.Size() == 3 }, time.Second, 10*time.Millisecond)

	assert.True(t, consume(dq, func(_ context.Context, el intRequest) error {
		assert.EqualValues(t, 3, el)
		return nil
	}))
	assert.Eventually(t, func() bool { return dq.Size() == 0 }, time.Second, 10*time.Millisecond)

	require.NoError(t, dq.Shutdown(context.Background()))
}

func TestDiskQueueOfferInvalidSize(t *testing.T) {
	dq := createTestDiskQueue(t, request.SizerTypeItems, 1)
	require.ErrorIs(t, dq.Offer(context.Background(), -1), errInvalidSize)
	require.NoError(t, dq.Shutdown(context.Background()))
}

func TestDiskQueueOfferZeroSize(t *testing.T) {
	dq := createTestDiskQueue(t, request.SizerTypeItems, 1)
	require.NoError(t, dq.Offer(context.Background(), 0))
	require.NoError(t, dq.Shutdown(context.Background()))
	// Because the size 0 is ignored, nothing to drain.
	assert.False(t, consume(dq, func(context.Context, intRequest) error { t.FailNow(); return nil }))
}

func TestDiskQueueRejectOverCapacityElements(t *testing.T) {
	dq := createTestDiskQueue(t, request.SizerTypeItems, 1)
	require.ErrorIs(t, dq.Offer(context.Background(), 8), errSizeTooLarge)
	require.NoError(t, dq.Shutdown(context.Background()))
}

func TestDiskQueueOverflow(t *testing.T) {
	dq := createTestDiskQueue(t, request.SizerTypeItems, 1)
	require.NoError(t, dq.Offer(context.Background(), 1))
	require.ErrorIs(t, dq.Offer(context.Background(), 1), ErrQueueIsFull)
	require.NoError(t, dq.Shutdown(context.Background()))
}

func TestDiskQueueDrainWhenShutdown(t *testing.T) {
	dq := createTestDiskQueue(t, request.SizerTypeItems, 7)
	require.NoError(t, dq.Offer(context.Background(), 1))
	require.NoError(t, dq.Offer(context.Background(), 3))

	assert.True(t, consume(dq, func(_ context.Context, el intRequest) error {
		assert.EqualValues(t, 1, el)
		return nil
	}))
	require.NoError(t, dq.Shutdown(context.Background()))

	// The remaining item is still readable via a fresh queue instance pointed
	// at the same data directory (disk_queue does not drain in-process after
	// Shutdown, unlike the in-memory queue, since the backing client is closed).
	assert.False(t, consume(dq, func(context.Context, intRequest) error { t.FailNow(); return nil }))
}

func TestDiskQueueReadReturnsFalseAfterShutdown(t *testing.T) {
	dq := createTestDiskQueue(t, request.SizerTypeItems, 7)
	require.NoError(t, dq.Shutdown(context.Background()))

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _, _, ok := dq.Read(context.Background())
		assert.False(t, ok)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Read did not return promptly after Shutdown")
	}
}

func TestDiskQueueReadUnblocksOnConcurrentShutdown(t *testing.T) {
	dq := createTestDiskQueue(t, request.SizerTypeItems, 100)

	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		_, _, _, ok := dq.Read(context.Background())
		assert.False(t, ok)
	}()

	// Let the reader block on Peek() with nothing in the queue.
	time.Sleep(100 * time.Millisecond)
	require.NoError(t, dq.Shutdown(context.Background()))

	select {
	case <-readDone:
	case <-time.After(2 * time.Second):
		t.Fatal("Read did not unblock after a concurrent Shutdown")
	}
}

func TestDiskQueue_StopAfterBadStart(t *testing.T) {
	set, _ := newSettingsWithDiskStorage(request.SizerTypeItems, 10)
	dq := newDiskQueue[intRequest](set)
	// verify that stopping a un-start/started w/error queue does not panic
	assert.NoError(t, dq.Shutdown(context.Background()))
}

func TestDiskQueue_StartError(t *testing.T) {
	set, _ := newSettingsWithDiskStorage(request.SizerTypeItems, 10)
	dq := newDiskQueue[intRequest](set)
	host := hosttest.NewHost(map[component.ID]component.Component{})
	require.ErrorIs(t, dq.Start(context.Background(), host), errNoDiskAccessClient)
	require.NoError(t, dq.Shutdown(context.Background()))
}

func TestToDiskAccessClient(t *testing.T) {
	testCases := []struct {
		name          string
		numStorages   int
		wrongType     bool
		expectedError error
	}{
		{
			name:        "obtain disk access extension by name",
			numStorages: 1,
		},
		{
			name:          "fail on not existing extension",
			numStorages:   0,
			expectedError: errNoDiskAccessClient,
		},
		{
			name:          "invalid extension type",
			numStorages:   1,
			wrongType:     true,
			expectedError: errDiskAccessWrongExtensionType,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			storageID := component.MustNewID(diskqueueextension.TypeStr)
			extensions := map[component.ID]component.Component{}
			if tt.numStorages > 0 {
				if tt.wrongType {
					nopFactory := extensiontest.NewNopFactory()
					nopExt, err := nopFactory.Create(context.Background(), extensiontest.NewNopSettings(nopFactory.Type()), nopFactory.CreateDefaultConfig())
					require.NoError(t, err)
					extensions[storageID] = nopExt
				} else {
					extensions[storageID] = newDiskAccessExtensionForTest(t, t.TempDir())
				}
			}
			host := hosttest.NewHost(extensions)
			ownerID := component.MustNewID("foo_exporter")

			client, err := toDiskAccessClient(context.Background(), storageID, host, ownerID, pipeline.SignalTraces)

			if tt.expectedError != nil {
				require.ErrorIs(t, err, tt.expectedError)
				assert.Nil(t, client)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, client)
				require.NoError(t, client.Shutdown(context.Background()))
			}
		})
	}
}

func TestDiskQueueOfferMarshalErrorUnrefs(t *testing.T) {
	set, storageID := newSettingsWithDiskStorage(request.SizerTypeItems, 100)
	rc := &countingReferenceCounter{}
	set.ReferenceCounter = rc
	set.Encoding = failingEncoding{marshalErr: errors.New("marshal failed")}
	dq := newDiskQueue[intRequest](set).(*diskQueue[intRequest])
	ext := newDiskAccessExtensionForTest(t, t.TempDir())
	host := hosttest.NewHost(map[component.ID]component.Component{storageID: ext})
	require.NoError(t, dq.Start(context.Background(), host))
	t.Cleanup(func() { _ = dq.Shutdown(context.Background()) })

	require.Error(t, dq.Offer(context.Background(), 1))
	assert.EqualValues(t, 0, rc.load())
}

func TestDiskQueueOfferWriteErrorUnrefs(t *testing.T) {
	set, storageID := newSettingsWithDiskStorage(request.SizerTypeItems, 100)
	rc := &countingReferenceCounter{}
	set.ReferenceCounter = rc
	dq := newDiskQueue[intRequest](set).(*diskQueue[intRequest])
	dir := t.TempDir()
	ext := newDiskAccessExtensionForTest(t, dir)
	host := hosttest.NewHost(map[component.ID]component.Component{storageID: ext})
	require.NoError(t, dq.Start(context.Background(), host))
	t.Cleanup(func() { _ = dq.Shutdown(context.Background()) })

	// Make the data directory read-only so the first Write fails to open
	// its backing file, without shutting down the client (which would make
	// any further Write hang forever waiting on a writeLoop that has
	// already exited).
	require.NoError(t, os.Chmod(dir, 0o400)) // #nosec G302
	t.Cleanup(func() { _ = os.Chmod(dir, 0o600) })

	require.Error(t, dq.Offer(context.Background(), 1))
	assert.EqualValues(t, 0, rc.load())
}

func TestDiskQueueReadUnmarshalErrorDoesNotPanic(t *testing.T) {
	set, storageID := newSettingsWithDiskStorage(request.SizerTypeItems, 100)
	rc := &countingReferenceCounter{}
	set.ReferenceCounter = rc
	dq := newDiskQueue[intRequest](set).(*diskQueue[intRequest])
	dq.encoding = failingEncoding{unmarshalErr: errors.New("unmarshal failed")}
	ext := newDiskAccessExtensionForTest(t, t.TempDir())
	host := hosttest.NewHost(map[component.ID]component.Component{storageID: ext})
	require.NoError(t, dq.Start(context.Background(), host))
	t.Cleanup(func() { _ = dq.Shutdown(context.Background()) })

	require.NoError(t, dq.diskAccessClient.Write(xqueue.WriteOp{Payload: []byte("x"), Size: 1}))
	_, _, done, ok := dq.Read(context.Background())
	assert.False(t, ok)
	assert.Nil(t, done)
}

// countingReferenceCounter tracks the net number of outstanding references,
// so tests can assert Offer correctly balances Ref/Unref on failure paths.
type countingReferenceCounter struct {
	mu  sync.Mutex
	ref int64
}

func (c *countingReferenceCounter) Ref(intRequest) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ref++
}

func (c *countingReferenceCounter) Unref(intRequest) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ref--
}

func (c *countingReferenceCounter) load() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ref
}

// failingEncoding lets tests force Marshal/Unmarshal failures independently.
type failingEncoding struct {
	marshalErr   error
	unmarshalErr error
}

func (f failingEncoding) Marshal(ctx context.Context, val intRequest) ([]byte, error) {
	if f.marshalErr != nil {
		return nil, f.marshalErr
	}
	return int64Encoding{}.Marshal(ctx, val)
}

func (f failingEncoding) Unmarshal(b []byte) (context.Context, intRequest, error) {
	if f.unmarshalErr != nil {
		return context.Background(), 0, f.unmarshalErr
	}
	return int64Encoding{}.Unmarshal(b)
}

func BenchmarkDiskQueue(b *testing.B) {
	dq := createTestDiskQueue(b, request.SizerTypeRequests, 10000000)

	req := intRequest(100)

	b.ReportAllocs()

	for b.Loop() {
		for range 100 {
			require.NoError(b, dq.Offer(context.Background(), req))
		}
		for range 100 {
			require.True(b, consume(dq, func(context.Context, intRequest) error { return nil }))
		}
	}
	require.NoError(b, dq.Shutdown(context.Background()))
}
