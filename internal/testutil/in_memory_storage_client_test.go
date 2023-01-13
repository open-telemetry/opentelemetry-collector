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
// limitations under the License

package testutil

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/extension/experimental/storage"
)

const (
	mockKey = "key"
)

var (
	mockValue = []byte("value")
)

func getValueAndAssertEmpty(t *testing.T, c storage.Client) {
	val, err := c.Get(context.Background(), mockKey)
	require.NoError(t, err)
	require.Nil(t, val)
}

func TestMockStorageClient_Get_valueDoesNotExist(t *testing.T) {
	c := NewInMemoryStorageClient()
	getValueAndAssertEmpty(t, c)
}

func TestMockStorageClient_Set_success(t *testing.T) {
	c := NewInMemoryStorageClient()
	err := c.Set(context.Background(), mockKey, mockValue)
	require.NoError(t, err)

	val, err := c.Get(context.Background(), mockKey)
	require.NoError(t, err)
	require.Equal(t, mockValue, val)
}

func TestMockStorageClient_Delete_success(t *testing.T) {
	c := NewInMemoryStorageClient()
	err := c.Set(context.Background(), mockKey, mockValue)
	require.NoError(t, err)

	err = c.Delete(context.Background(), mockKey)
	require.NoError(t, err)

	getValueAndAssertEmpty(t, c)
}

func TestMockStorageClient_Close_success(t *testing.T) {
	c := NewInMemoryStorageClient()
	err := c.Close(context.Background())
	require.NoError(t, err)

	castedClient := c.(*InMemoryStorageClient)
	require.Equal(t, uint64(1), castedClient.GetCloseCount())
}

func TestMockStorageClient_Batch_success(t *testing.T) {
	c := NewInMemoryStorageClient()

	setOp := storage.SetOperation(mockKey, mockValue)
	getOpExist := storage.GetOperation(mockKey)
	deleteOp := storage.DeleteOperation(mockKey)
	getOpEmpty := storage.GetOperation(mockKey)

	err := c.Batch(context.Background(), setOp, getOpExist, deleteOp, getOpEmpty)
	require.NoError(t, err)

	require.Equal(t, mockValue, getOpExist.Value)
	require.Nil(t, getOpEmpty.Value)
}
