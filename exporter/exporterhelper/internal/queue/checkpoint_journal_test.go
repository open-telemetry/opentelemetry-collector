// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/extension/xextension/storage"
)

type checkpointFaultClient struct {
	storage.Client
	getKey, setKey string
	failure        error
}

func (c *checkpointFaultClient) Get(ctx context.Context, key string) ([]byte, error) {
	if key == c.getKey {
		return nil, c.failure
	}
	return c.Client.Get(ctx, key)
}

func (c *checkpointFaultClient) Set(ctx context.Context, key string, value []byte) error {
	if key == c.setKey {
		return c.failure
	}
	return c.Client.Set(ctx, key, value)
}

func TestCheckpointJournalRecoversPartialEnvelopePublication(t *testing.T) {
	ctx := context.Background()
	client := &checkpointFaultClient{Client: newFakeBoundedStorageClient(1 << 20), failure: errors.New("disk failure")}
	pq := newPersistentQueue[intRequest](newSettingsWithStorage(request.SizerTypeRequests, 10)).(*persistentQueue[intRequest])
	pq.initClient(ctx, client)
	require.NoError(t, client.Set(ctx, "1", []byte("old-A")))
	require.NoError(t, client.Set(ctx, "2", []byte("old-B")))
	client.setKey = "2"
	require.NoError(t, pq.SaveCheckpointAndItems(ctx, "ordered", []byte("tail-B"), []request.QueueItemUpdate{{Token: 1, Value: []byte("retired-A")}, {Token: 2, Value: []byte("retired-B")}}))
	// Only the first envelope was copied. The single committed journal still
	// contains both retirements and their matching tail.
	body, err := client.Get(ctx, "2")
	require.NoError(t, err)
	require.Equal(t, "old-B", string(body))
	_, _, err = pq.LoadCheckpoint(ctx, "ordered")
	require.ErrorIs(t, err, client.failure)
	client.setKey = ""
	restarted := newPersistentQueue[intRequest](newSettingsWithStorage(request.SizerTypeRequests, 10)).(*persistentQueue[intRequest])
	restarted.initClient(ctx, client)
	tail, found, err := restarted.LoadCheckpoint(ctx, "ordered")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "tail-B", string(tail))
	body, err = restarted.LoadQueueItem(ctx, 2)
	require.NoError(t, err)
	require.Equal(t, "retired-B", string(body))
}

func TestCheckpointReadErrorDoesNotBecomeMissingCheckpoint(t *testing.T) {
	ctx := context.Background()
	client := &checkpointFaultClient{Client: newFakeBoundedStorageClient(1 << 20), failure: errors.New("disk I/O failure")}
	pq := newPersistentQueue[intRequest](newSettingsWithStorage(request.SizerTypeRequests, 10)).(*persistentQueue[intRequest])
	pq.initClient(ctx, client)
	client.getKey = "ocp/ordered"
	_, found, err := pq.LoadCheckpoint(ctx, "ordered")
	require.ErrorIs(t, err, client.failure)
	require.False(t, found)
}

func TestReplayLoadFailurePreservesDispatchedOwnership(t *testing.T) {
	ctx := context.Background()
	client := &checkpointFaultClient{Client: newFakeBoundedStorageClient(1 << 20), failure: errors.New("disk I/O failure")}
	metadata := &PersistentMetadata{ReadIndex: 1, WriteIndex: 1, CurrentlyDispatchedItems: []uint64{0}, RequestsSize: 1}
	body, err := proto.Marshal(metadata)
	require.NoError(t, err)
	require.NoError(t, client.Set(ctx, metadataKey, body))
	client.getKey = replayItemsKey
	set := newSettingsWithStorage(request.SizerTypeRequests, 10)
	set.ReplayInOrder = true
	pq := newPersistentQueue[intRequest](set).(*persistentQueue[intRequest])
	pq.initClient(ctx, client)
	require.ErrorIs(t, pq.startupErr, client.failure)
	require.Equal(t, []uint64{0}, pq.metadata.CurrentlyDispatchedItems)
}
