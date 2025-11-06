// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlphttpexporter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/client"
)

func TestMetadataKeysPartitioner_MergeCtx(t *testing.T) {
	t.Run("merge contexts with same metadata key values", func(t *testing.T) {
		partitioner := metadataKeysPartitioner{keys: []string{"key1", "key2"}}

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

		mergedCtx := partitioner.MergeCtx(ctx1, ctx2)
		require.NotNil(t, mergedCtx)

		mergedMeta := client.FromContext(mergedCtx).Metadata
		assert.Equal(t, []string{"value1"}, mergedMeta.Get("key1"))
		assert.Equal(t, []string{"value2"}, mergedMeta.Get("key2"))
	})

	t.Run("merge contexts with same metadata key values and additional keys", func(t *testing.T) {
		partitioner := metadataKeysPartitioner{keys: []string{"key1"}}

		ctx1 := client.NewContext(
			context.Background(),
			client.Info{
				Metadata: client.NewMetadata(map[string][]string{
					"key1": {"value1"},
					"other": {"other1"},
				}),
			},
		)

		ctx2 := client.NewContext(
			context.Background(),
			client.Info{
				Metadata: client.NewMetadata(map[string][]string{
					"key1": {"value1"},
					"other": {"other2"},
				}),
			},
		)

		mergedCtx := partitioner.MergeCtx(ctx1, ctx2)
		require.NotNil(t, mergedCtx)

		mergedMeta := client.FromContext(mergedCtx).Metadata
		assert.Equal(t, []string{"value1"}, mergedMeta.Get("key1"))
		// Other keys should not be in merged metadata since they're not in partitioner keys
		assert.Empty(t, mergedMeta.Get("other"))
	})

	t.Run("merge contexts with empty metadata", func(t *testing.T) {
		partitioner := metadataKeysPartitioner{keys: []string{"key1"}}

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

		mergedCtx := partitioner.MergeCtx(ctx1, ctx2)
		require.NotNil(t, mergedCtx)

		mergedMeta := client.FromContext(mergedCtx).Metadata
		assert.Empty(t, mergedMeta.Get("key1"))
	})

	t.Run("panic when one context has metadata and other is empty", func(t *testing.T) {
		partitioner := metadataKeysPartitioner{keys: []string{"key1"}}

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

		// When one has value and other is empty, they are different, so it should panic
		assert.Panics(t, func() {
			partitioner.MergeCtx(ctx1, ctx2)
		}, "should panic when contexts have different metadata values")
	})

	t.Run("panic when contexts have different metadata key values", func(t *testing.T) {
		partitioner := metadataKeysPartitioner{keys: []string{"key1"}}

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

		assert.Panics(t, func() {
			partitioner.MergeCtx(ctx1, ctx2)
		}, "should panic when contexts have different metadata values")
	})

	t.Run("panic when contexts have different metadata key values for multiple keys", func(t *testing.T) {
		partitioner := metadataKeysPartitioner{keys: []string{"key1", "key2"}}

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

		assert.Panics(t, func() {
			partitioner.MergeCtx(ctx1, ctx2)
		}, "should panic when contexts have different metadata values")
	})

	t.Run("merge contexts with multiple values for same key", func(t *testing.T) {
		partitioner := metadataKeysPartitioner{keys: []string{"key1"}}

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

		mergedCtx := partitioner.MergeCtx(ctx1, ctx2)
		require.NotNil(t, mergedCtx)

		mergedMeta := client.FromContext(mergedCtx).Metadata
		assert.Equal(t, []string{"value1", "value2"}, mergedMeta.Get("key1"))
	})

	t.Run("panic when contexts have different multiple values for same key", func(t *testing.T) {
		partitioner := metadataKeysPartitioner{keys: []string{"key1"}}

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

		assert.Panics(t, func() {
			partitioner.MergeCtx(ctx1, ctx2)
		}, "should panic when contexts have different metadata values")
	})
}

