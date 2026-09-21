// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package diskqueueextension // import "go.opentelemetry.io/collector/extension/diskqueueextension"

import "time"

type Config struct {
	DataPath              string        `mapstructure:"data_path"`
	MaxBytesPerFile       int64         `mapstructure:"max_bytes_per_file"`
	SyncEvery             int64         `mapstructure:"sync_every"`
	SyncTimeout           time.Duration `mapstructure:"sync_timeout"`
	MetadataTruncateEvery int           `mapstructure:"truncate_every"`
}
