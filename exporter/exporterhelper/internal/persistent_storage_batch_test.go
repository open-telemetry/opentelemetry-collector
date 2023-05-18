// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPersistentStorageBatch_Operations(t *testing.T) {
	path := t.TempDir()

	ext := createStorageExtension(path)
	client := createTestClient(ext)
	ps := createTestPersistentStorage(client)

	itemIndexValue := itemIndex(123)
	itemIndexArrayValue := []itemIndex{itemIndex(1), itemIndex(2)}

	_, err := newBatch(ps).
		setItemIndex("index", itemIndexValue).
		setItemIndexArray("arr", itemIndexArrayValue).
		execute(context.Background())
	require.NoError(t, err)

	batch, err := newBatch(ps).
		get("index", "arr").
		execute(context.Background())
	require.NoError(t, err)

	retrievedItemIndexValue, err := batch.getItemIndexResult("index")
	require.NoError(t, err)
	assert.Equal(t, itemIndexValue, retrievedItemIndexValue)

	retrievedItemIndexArrayValue, err := batch.getItemIndexArrayResult("arr")
	require.NoError(t, err)
	assert.Equal(t, itemIndexArrayValue, retrievedItemIndexArrayValue)

	_, err = newBatch(ps).delete("index", "arr").execute(context.Background())
	require.NoError(t, err)

	batch, err = newBatch(ps).
		get("index", "arr").
		execute(context.Background())
	require.NoError(t, err)

	_, err = batch.getItemIndexResult("index")
	assert.Error(t, err, errValueNotSet)

	retrievedItemIndexArrayValue, err = batch.getItemIndexArrayResult("arr")
	require.NoError(t, err)
	assert.Nil(t, retrievedItemIndexArrayValue)
}
