// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package persistentqueue // import "go.opentelemetry.io/collector/exporter/exporterhelper/queue/persistentqueue"

import (
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
	"go.opentelemetry.io/collector/exporter/exporterhelper/queue"
	"go.opentelemetry.io/collector/exporter/exporterhelper/queue/memoryqueue"
	"go.opentelemetry.io/collector/exporter/exporterhelper/request"
)

type factory struct {
	cfg         Config
	marshaler   request.Marshaler
	unmarshaler request.Unmarshaler
}

// NewFactory returns a factory of persistent queue.
// If cfg.StorageID is nil then it falls back to memory queue.
// This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func NewFactory(cfg Config, marshaler request.Marshaler, unmarshaler request.Unmarshaler) queue.Factory {
	if cfg.StorageID == nil {
		return memoryqueue.NewFactory(memoryqueue.Config{Config: cfg.Config})
	}
	return &factory{cfg: cfg, marshaler: marshaler, unmarshaler: unmarshaler}
}

// Create creates a persistent queue based on the given settings.
// This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func (f *factory) Create(set queue.Settings) queue.Queue {
	if !f.cfg.Enabled {
		return nil
	}
	return internal.NewPersistentQueue(f.cfg.QueueSize, f.cfg.NumConsumers, *f.cfg.StorageID, set, f.marshaler, f.unmarshaler)
}

var _ queue.Factory = (*factory)(nil)
