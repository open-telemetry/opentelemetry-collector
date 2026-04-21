// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/client"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"
)

func TestMetadataKeysPartitioner_MergeCtx(t *testing.T) {
	t.Run("merge contexts with same metadata key values", func(t *testing.T) {
		mergeCtx := NewMetadataKeysMergeCtx([]string{"key1", "key2"})

		ctx1 := client.NewContext(
			context.Background(),
			client.Info{
				Metadata: client.NewMetadata(map[string][]string{
					"key1": {"value1"},
					"key2": {"value2"},
				}),
			},
		)

		ctx2 := client.NewContext(
			context.Background(),
			client.Info{
				Metadata: client.NewMetadata(map[string][]string{
					"key1": {"value1"},
					"key2": {"value2"},
				}),
			},
		)

		mergedCtx := mergeCtx(ctx1, ctx2)
		require.NotNil(t, mergedCtx)

		mergedMeta := client.FromContext(mergedCtx).Metadata
		assert.Equal(t, []string{"value1"}, mergedMeta.Get("key1"))
		assert.Equal(t, []string{"value2"}, mergedMeta.Get("key2"))
	})

	t.Run("merge contexts with same metadata key values and additional keys", func(t *testing.T) {
		mergeCtx := NewMetadataKeysMergeCtx([]string{"key1"})

		ctx1 := client.NewContext(
			context.Background(),
			client.Info{
				Metadata: client.NewMetadata(map[string][]string{
					"key1":  {"value1"},
					"other": {"other1"},
				}),
			},
		)

		ctx2 := client.NewContext(
			context.Background(),
			client.Info{
				Metadata: client.NewMetadata(map[string][]string{
					"key1":  {"value1"},
					"other": {"other2"},
				}),
			},
		)

		mergedCtx := mergeCtx(ctx1, ctx2)
		require.NotNil(t, mergedCtx)

		mergedMeta := client.FromContext(mergedCtx).Metadata
		assert.Equal(t, []string{"value1"}, mergedMeta.Get("key1"))
		// Other keys should not be in merged metadata since they're not in partitioner keys
		assert.Empty(t, mergedMeta.Get("other"))
	})

	t.Run("merge contexts with empty metadata", func(t *testing.T) {
		mergeCtx := NewMetadataKeysMergeCtx([]string{"key1"})

		ctx1 := client.NewContext(
			context.Background(),
			client.Info{
				Metadata: client.NewMetadata(map[string][]string{}),
			},
		)

		ctx2 := client.NewContext(
			context.Background(),
			client.Info{
				Metadata: client.NewMetadata(map[string][]string{}),
			},
		)

		mergedCtx := mergeCtx(ctx1, ctx2)
		require.NotNil(t, mergedCtx)

		mergedMeta := client.FromContext(mergedCtx).Metadata
		assert.Empty(t, mergedMeta.Get("key1"))
	})

	t.Run("merge when one context has metadata and other is empty", func(t *testing.T) {
		mergeCtx := NewMetadataKeysMergeCtx([]string{"key1"})

		ctx1 := client.NewContext(
			context.Background(),
			client.Info{
				Metadata: client.NewMetadata(map[string][]string{
					"key1": {"value1"},
				}),
			},
		)

		ctx2 := client.NewContext(
			context.Background(),
			client.Info{
				Metadata: client.NewMetadata(map[string][]string{}),
			},
		)

		mergedCtx := mergeCtx(ctx1, ctx2)
		require.NotNil(t, mergedCtx)
		mergedMeta := client.FromContext(mergedCtx).Metadata
		assert.Equal(t, []string{"value1"}, mergedMeta.Get("key1"))
	})

	t.Run("merge when contexts have different metadata key values", func(t *testing.T) {
		mergeCtx := NewMetadataKeysMergeCtx([]string{"key1"})

		ctx1 := client.NewContext(
			context.Background(),
			client.Info{
				Metadata: client.NewMetadata(map[string][]string{
					"key1": {"value1"},
				}),
			},
		)

		ctx2 := client.NewContext(
			context.Background(),
			client.Info{
				Metadata: client.NewMetadata(map[string][]string{
					"key1": {"value2"},
				}),
			},
		)

		mergedCtx := mergeCtx(ctx1, ctx2)
		require.NotNil(t, mergedCtx)
		mergedMeta := client.FromContext(mergedCtx).Metadata
		assert.Equal(t, []string{"value1"}, mergedMeta.Get("key1"))
	})

	t.Run("merge when contexts have different metadata key values for multiple keys", func(t *testing.T) {
		mergeCtx := NewMetadataKeysMergeCtx([]string{"key1", "key2"})

		ctx1 := client.NewContext(
			context.Background(),
			client.Info{
				Metadata: client.NewMetadata(map[string][]string{
					"key1": {"value1"},
					"key2": {"value2"},
				}),
			},
		)

		ctx2 := client.NewContext(
			context.Background(),
			client.Info{
				Metadata: client.NewMetadata(map[string][]string{
					"key1": {"value1"},
					"key2": {"different"},
				}),
			},
		)

		mergedCtx := mergeCtx(ctx1, ctx2)
		require.NotNil(t, mergedCtx)
		mergedMeta := client.FromContext(mergedCtx).Metadata
		assert.Equal(t, []string{"value1"}, mergedMeta.Get("key1"))
		assert.Equal(t, []string{"value2"}, mergedMeta.Get("key2"))
	})

	t.Run("merge contexts with multiple values for same key", func(t *testing.T) {
		mergeCtx := NewMetadataKeysMergeCtx([]string{"key1"})

		ctx1 := client.NewContext(
			context.Background(),
			client.Info{
				Metadata: client.NewMetadata(map[string][]string{
					"key1": {"value1", "value2"},
				}),
			},
		)

		ctx2 := client.NewContext(
			context.Background(),
			client.Info{
				Metadata: client.NewMetadata(map[string][]string{
					"key1": {"value1", "value2"},
				}),
			},
		)

		mergedCtx := mergeCtx(ctx1, ctx2)
		require.NotNil(t, mergedCtx)

		mergedMeta := client.FromContext(mergedCtx).Metadata
		assert.Equal(t, []string{"value1", "value2"}, mergedMeta.Get("key1"))
	})

	t.Run("merge when contexts have different multiple values for same key", func(t *testing.T) {
		mergeCtx := NewMetadataKeysMergeCtx([]string{"key1"})

		ctx1 := client.NewContext(
			context.Background(),
			client.Info{
				Metadata: client.NewMetadata(map[string][]string{
					"key1": {"value1", "value2"},
				}),
			},
		)

		ctx2 := client.NewContext(
			context.Background(),
			client.Info{
				Metadata: client.NewMetadata(map[string][]string{
					"key1": {"value1", "value3"},
				}),
			},
		)

		mergedCtx := mergeCtx(ctx1, ctx2)
		require.NotNil(t, mergedCtx)
		mergedMeta := client.FromContext(mergedCtx).Metadata
		assert.Equal(t, []string{"value1", "value2"}, mergedMeta.Get("key1"))
	})
}

func TestMetadataKeysPartitioner_GetKey(t *testing.T) {
	t.Run("single key with single value", func(t *testing.T) {
		partitioner := NewMetadataKeysPartitioner([]string{"key1"})

		ctx := client.NewContext(
			context.Background(),
			client.Info{
				Metadata: client.NewMetadata(map[string][]string{
					"key1": {"value1"},
				}),
			},
		)

		key := partitioner.GetKey(ctx, &requesttest.FakeRequest{})
		expected := "key1\x00value1"
		assert.Equal(t, expected, key)
	})

	t.Run("single key with multiple values", func(t *testing.T) {
		partitioner := NewMetadataKeysPartitioner([]string{"key1"})

		ctx := client.NewContext(
			context.Background(),
			client.Info{
				Metadata: client.NewMetadata(map[string][]string{
					"key1": {"value1", "value2", "value3"},
				}),
			},
		)

		key := partitioner.GetKey(ctx, &requesttest.FakeRequest{})
		// Format: key1\0value1\0value2\0value3
		expected := "key1\x00value1\x00value2\x00value3"
		assert.Equal(t, expected, key)
	})

	t.Run("multiple keys with values", func(t *testing.T) {
		partitioner := NewMetadataKeysPartitioner([]string{"key1", "key2"})

		ctx := client.NewContext(
			context.Background(),
			client.Info{
				Metadata: client.NewMetadata(map[string][]string{
					"key1": {"value1"},
					"key2": {"value2"},
				}),
			},
		)

		key := partitioner.GetKey(ctx, &requesttest.FakeRequest{})
		// Format: key1\0value1\0key2\0value2 (separator between keys)
		expected := "key1\x00value1\x00key2\x00value2"
		assert.Equal(t, expected, key)
	})

	t.Run("multiple keys with multiple values", func(t *testing.T) {
		partitioner := NewMetadataKeysPartitioner([]string{"key1", "key2"})

		ctx := client.NewContext(
			context.Background(),
			client.Info{
				Metadata: client.NewMetadata(map[string][]string{
					"key1": {"value1", "value1b"},
					"key2": {"value2", "value2b"},
				}),
			},
		)

		key := partitioner.GetKey(ctx, &requesttest.FakeRequest{})
		// Format: key1\0value1\0value1b\0key2\0value2\0value2b
		expected := "key1\x00value1\x00value1b\x00key2\x00value2\x00value2b"
		assert.Equal(t, expected, key)
	})

	t.Run("keys that don't exist in metadata are skipped", func(t *testing.T) {
		partitioner := NewMetadataKeysPartitioner([]string{"key1", "key2", "key3"})

		ctx := client.NewContext(
			context.Background(),
			client.Info{
				Metadata: client.NewMetadata(map[string][]string{
					"key1": {"value1"},
					// key2 is missing
					"key3": {"value3"},
				}),
			},
		)

		key := partitioner.GetKey(ctx, &requesttest.FakeRequest{})
		// Format: key1\0value1\0key3\0value3 (key2 is skipped)
		expected := "key1\x00value1\x00key3\x00value3"
		assert.Equal(t, expected, key)
	})

	t.Run("empty metadata returns empty string", func(t *testing.T) {
		partitioner := NewMetadataKeysPartitioner([]string{"key1", "key2"})

		ctx := client.NewContext(
			context.Background(),
			client.Info{
				Metadata: client.NewMetadata(map[string][]string{}),
			},
		)

		key := partitioner.GetKey(ctx, &requesttest.FakeRequest{})
		assert.Empty(t, key)
	})

	t.Run("keys with empty values are skipped", func(t *testing.T) {
		partitioner := NewMetadataKeysPartitioner([]string{"key1", "key2", "key3"})

		ctx := client.NewContext(
			context.Background(),
			client.Info{
				Metadata: client.NewMetadata(map[string][]string{
					"key1": {"value1"},
					"key2": {}, // empty slice
					"key3": {"value3"},
				}),
			},
		)

		key := partitioner.GetKey(ctx, &requesttest.FakeRequest{})
		// Format: key1\0value1\0key3\0value3 (key2 is skipped)
		expected := "key1\x00value1\x00key3\x00value3"
		assert.Equal(t, expected, key)
	})

	t.Run("keys in order respect partitioner key order", func(t *testing.T) {
		partitioner := NewMetadataKeysPartitioner([]string{"key3", "key1", "key2"})

		ctx := client.NewContext(
			context.Background(),
			client.Info{
				Metadata: client.NewMetadata(map[string][]string{
					"key1": {"value1"},
					"key2": {"value2"},
					"key3": {"value3"},
				}),
			},
		)

		key := partitioner.GetKey(ctx, &requesttest.FakeRequest{})
		// Format should follow partitioner order: key3\0value3\0key1\0value1\0key2\0value2
		expected := "key3\x00value3\x00key1\x00value1\x00key2\x00value2"
		assert.Equal(t, expected, key)
	})

	t.Run("additional metadata keys not in partitioner are ignored", func(t *testing.T) {
		partitioner := NewMetadataKeysPartitioner([]string{"key1"})

		ctx := client.NewContext(
			context.Background(),
			client.Info{
				Metadata: client.NewMetadata(map[string][]string{
					"key1":  {"value1"},
					"other": {"other1"},
					"extra": {"extra1"},
				}),
			},
		)

		key := partitioner.GetKey(ctx, &requesttest.FakeRequest{})
		// Format: key1\0value1 (other keys are ignored)
		expected := "key1\x00value1"
		assert.Equal(t, expected, key)
	})

	t.Run("returns nil partitioner when keys are empty", func(t *testing.T) {
		partitioner := NewMetadataKeysPartitioner([]string{})
		assert.Nil(t, partitioner)
	})

	t.Run("returns nil merge function when keys are empty", func(t *testing.T) {
		mergeCtx := NewMetadataKeysMergeCtx([]string{})
		assert.Nil(t, mergeCtx)
	})
}
