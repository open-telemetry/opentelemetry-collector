// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package diskqueueextension

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/confmap"
)

func TestUnmarshalConfig(t *testing.T) {
	cm := confmap.NewFromStringMap(map[string]any{
		"data_path":          "/tmp/queue",
		"max_bytes_per_file": 2048,
		"sync_every":         5,
		"sync_timeout":       "2s",
		"truncate_every":     500,
	})
	cfg := createDefaultConfig()
	require.NoError(t, cm.Unmarshal(&cfg))
	assert.Equal(t, &Config{
		DataPath:              "/tmp/queue",
		MaxBytesPerFile:       2048,
		SyncEvery:             5,
		SyncTimeout:           2 * time.Second,
		MetadataTruncateEvery: 500,
	}, cfg)
}

func TestUnmarshalDefaultConfig(t *testing.T) {
	cfg := createDefaultConfig()
	require.NoError(t, confmap.New().Unmarshal(&cfg))
	assert.Equal(t, createDefaultConfig(), cfg)
}
