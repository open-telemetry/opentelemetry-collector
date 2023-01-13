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
