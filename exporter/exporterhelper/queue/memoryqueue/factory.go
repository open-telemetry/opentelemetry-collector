// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package memoryqueue // import "go.opentelemetry.io/collector/exporter/exporterhelper/queue/memoryqueue"

import (
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
	"go.opentelemetry.io/collector/exporter/exporterhelper/queue"
)

type factory struct {
	cfg Config
}

// NewFactory returns a factory of memory queue.
// This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func NewFactory(cfg Config) queue.Factory {
	return &factory{cfg: cfg}
}

// Create creates a memory queue based on the given settings.
// This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func (f *factory) Create(set queue.Settings) queue.Queue {
	if !f.cfg.Enabled {
		return nil
	}
	return internal.NewBoundedMemoryQueue(f.cfg.QueueSize, f.cfg.NumConsumers, set.Callback)
}

var _ queue.Factory = (*factory)(nil)
