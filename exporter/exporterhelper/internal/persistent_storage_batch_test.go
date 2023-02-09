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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/extension/experimental/storage"
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

func TestPersistentStorageBatch_Errors(t *testing.T) {
	path := t.TempDir()

	ext := createStorageExtension(path)
	client := createTestClient(ext)
	ps := createTestPersistentStorage(client)

	marshal := func(_ interface{}) ([]byte, error) {
		return nil, errors.New("marshal error")
	}
	core, observed := observer.New(zap.DebugLevel)
	ps.logger = zap.New(core)
	batch := newBatch(ps).set("marshal-error", 1, marshal)
	require.Len(t, observed.FilterLevelExact(zap.DebugLevel).TakeAll(), 1)

	invalidOperation := storage.GetOperation("invalid")
	invalidOperation.Type = -1
	batch.operations = append(batch.operations, invalidOperation)
	_, err := batch.execute(context.Background())
	require.Error(t, err)

	_, err = batch.getRequestResult("not-present")
	require.Equal(t, errKeyNotPresentInBatch, err)

	batch = batch.get("value-not-set")
	_, err = batch.getRequestResult("value-not-set")
	require.Equal(t, errValueNotSet, err)
}
