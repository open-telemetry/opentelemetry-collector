// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch

import (
	"context"
	"strconv"
	"testing"

	"go.opentelemetry.io/collector/client"
)

var metadataPartitionKeyBenchmarkSink string

func BenchmarkMetadataKeysPartitioner_GetKey(b *testing.B) {
	keys := []string{"key1", "key2", "key3", "key4"}
	for n := range 5 {
		b.Run(strconv.Itoa(n), func(b *testing.B) {
			metadata := make(map[string][]string, n)
			for _, key := range keys[:n] {
				metadata[key] = []string{"value"}
			}
			ctx := client.NewContext(context.Background(), client.Info{Metadata: client.NewMetadata(metadata)})
			p := metadataKeysPartitioner{keys: keys[:n]}
			b.ReportAllocs()
			for b.Loop() {
				metadataPartitionKeyBenchmarkSink = p.GetKey(ctx, nil)
			}
		})
	}
}
