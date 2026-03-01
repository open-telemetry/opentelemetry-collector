// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"

import (
	"bytes"
	"context"

	"go.opentelemetry.io/collector/client"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
)

// metadataKeysPartitioner partitions requests based on client metadata keys.
type metadataKeysPartitioner struct {
	keys []string
}

// NewMetadataKeysPartitioner creates a new partitioner that partitions requests
// based on the specified metadata keys. If keys is empty, returns nil.
func NewMetadataKeysPartitioner(keys []string) Partitioner[request.Request] {
	if len(keys) == 0 {
		return nil
	}
	return &metadataKeysPartitioner{keys: keys}
}

// GetKey returns a partition key based on the metadata keys configured.
func (p *metadataKeysPartitioner) GetKey(
	ctx context.Context,
	_ request.Request,
) string {
	var kb bytes.Buffer
	meta := client.FromContext(ctx).Metadata

	var afterFirst bool
	for _, k := range p.keys {
		if values := meta.Get(k); len(values) != 0 {
			if afterFirst {
				kb.WriteByte(0)
			}
			kb.WriteString(k)
			afterFirst = true
			for _, val := range values {
				kb.WriteByte(0)
				kb.WriteString(val)
			}
		}
	}
	return kb.String()
}

// NewMetadataKeysMergeCtx creates a merge function for contexts that merges
// metadata based on the specified keys. If keys is empty, returns nil.
func NewMetadataKeysMergeCtx(keys []string) func(context.Context, context.Context) context.Context {
	if len(keys) == 0 {
		return nil
	}
	return func(ctx1, _ context.Context) context.Context {
		m1 := client.FromContext(ctx1).Metadata

		m := make(map[string][]string, len(keys))
		for _, key := range keys {
			v1 := m1.Get(key)
			if len(v1) > 0 {
				m[key] = v1
			}
		}
		return client.NewContext(
			context.Background(),
			client.Info{Metadata: client.NewMetadata(m)},
		)
	}
}
