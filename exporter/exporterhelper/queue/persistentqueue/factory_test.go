// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package persistentqueue

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterhelper/queue"
)

func TestNewFactory(t *testing.T) {
	cfg := NewDefaultConfig()
	f := NewFactory(cfg, nil, nil)
	_, ok := f.(*factory) // should be memory queue factory if storage isn't set
	assert.False(t, ok)

	storage := component.NewID("fake-storage")
	cfg.StorageID = &storage
	f = NewFactory(cfg, nil, nil)
	_, ok = f.(*factory)
	assert.True(t, ok)
	assert.NotNil(t, f.Create(queue.Settings{}))

	cfg.Enabled = false
	f = NewFactory(cfg, nil, nil)
	assert.Nil(t, f.Create(queue.Settings{}))
}
