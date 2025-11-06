// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlphttpexporter

import (
	"bytes"
	"context"
	"fmt"
	"slices"

	"go.opentelemetry.io/collector/client"
	"go.opentelemetry.io/collector/exporter/exporterhelper/xexporterhelper"
)

type metadataKeysPartitioner struct {
	keys []string
}

func (p metadataKeysPartitioner) GetKey(
	ctx context.Context,
	_ xexporterhelper.Request,
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

func (p metadataKeysPartitioner) MergeCtx(
	ctx1, ctx2 context.Context,
) context.Context {
	m1 := client.FromContext(ctx1).Metadata
	m2 := client.FromContext(ctx2).Metadata

	m := make(map[string][]string, len(p.keys))
	for _, key := range p.keys {
		v1 := m1.Get(key)
		v2 := m2.Get(key)
		if len(v1) == 0 && len(v2) == 0 {
			continue
		}

		// Since the mergeCtx is based on partition key, we MUST have the same
		// partition key-values in both the metadata. If they are not same then
		// fail fast and dramatically.
		if !slices.Equal(v1, v2) {
			panic(fmt.Errorf(
				"unexpected client metadata found when merging context for key %s", key,
			))
		}
		m[key] = v1
	}
	return client.NewContext(
		context.Background(),
		client.Info{Metadata: client.NewMetadata(m)},
	)
}
