// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package memoryqueue

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/exporter/exporterhelper/queue"
)

func TestNewFactory(t *testing.T) {
	cfg := NewDefaultConfig()
	f := NewFactory(cfg)
	assert.NotNil(t, f.Create(queue.Settings{}))

	cfg.Enabled = false
	f = NewFactory(cfg)
	assert.Nil(t, f.Create(queue.Settings{}))
}
