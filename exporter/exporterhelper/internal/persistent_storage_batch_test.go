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
