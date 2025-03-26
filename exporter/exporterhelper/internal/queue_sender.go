// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"
	"errors"
	"fmt"
	"math"
	"runtime"
	"time"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
)

// QueueBatchSettings is a subset of the queuebatch.Settings that are needed when used within an Exporter.
type QueueBatchSettings[K any] struct {
	Encoding queuebatch.Encoding[K]
	Sizers   map[request.SizerType]request.Sizer[K]
}

// NewDefaultQueueConfig returns the default config for QueueConfig.
// By default, the queue stores 1000 items of telemetry and is non-blocking when full.
func NewDefaultQueueConfig() QueueConfig {
	return QueueConfig{
		Enabled:      true,
		Sizer:        request.SizerTypeRequests,
		NumConsumers: 10,
		// By default, batches are 8192 spans, for a total of up to 8 million spans in the queue
		// This can be estimated at 1-4 GB worth of maximum memory usage
		// This default is probably still too high, and may be adjusted further down in a future release
		QueueSize:       1_000,
		BlockOnOverflow: false,
	}
}

// QueueConfig defines configuration for queueing requests before exporting.
// It's supposed to be used with the new exporter helpers New[Traces|Metrics|Logs]RequestExporter.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type QueueConfig struct {
	// Enabled indicates whether to not enqueue batches before exporting.
	Enabled bool `mapstructure:"enabled"`

	// WaitForResult determines if incoming requests are blocked until the request is processed or not.
	// Currently, this option is not available when persistent queue is configured using the storage configuration.
	WaitForResult bool `mapstructure:"wait_for_result"`

	// Sizer determines the type of size measurement used by this component.
	// It accepts "requests", "items", or "bytes".
	Sizer request.SizerType `mapstructure:"sizer"`

	// QueueSize represents the maximum data size allowed for concurrent storage and processing.
	QueueSize int `mapstructure:"queue_size"`

	// NumConsumers is the number of consumers from the queue.
	NumConsumers int `mapstructure:"num_consumers"`

	// Deprecated: [v0.123.0] use `block_on_overflow`.
	Blocking bool `mapstructure:"blocking"`

	// BlockOnOverflow determines the behavior when the component's QueueSize limit is reached.
	// If true, the component will wait for space; otherwise, operations will immediately return a retryable error.
	BlockOnOverflow bool `mapstructure:"block_on_overflow"`

	// StorageID if not empty, enables the persistent storage and uses the component specified
	// as a storage extension for the persistent queue
	StorageID *component.ID `mapstructure:"storage"`

	hasBlocking bool
}

func (qCfg *QueueConfig) Unmarshal(conf *confmap.Conf) error {
	if err := conf.Unmarshal(qCfg); err != nil {
		return err
	}

	// If user still uses the old blocking, override and will log error during initialization.
	if conf.IsSet("blocking") {
		qCfg.hasBlocking = true
		qCfg.BlockOnOverflow = qCfg.Blocking
	}

	return nil
}

// Validate checks if the Config is valid
func (qCfg *QueueConfig) Validate() error {
	if !qCfg.Enabled {
		return nil
	}

	if qCfg.NumConsumers <= 0 {
		return errors.New("`num_consumers` must be positive")
	}

	if qCfg.QueueSize <= 0 {
		return errors.New("`queue_size` must be positive")
	}

	if qCfg.StorageID != nil && qCfg.WaitForResult {
		return errors.New("`wait_for_result` is not supported with a persistent queue configured with `storage`")
	}

	// Only support request sizer for persistent queue at this moment.
	if qCfg.StorageID != nil && qCfg.Sizer != request.SizerTypeRequests {
		return errors.New("persistent queue configured with `storage` only supports `requests` sizer")
	}
	return nil
}

func NewQueueSender(
	qSet queuebatch.Settings[request.Request],
	qCfg QueueConfig,
	bCfg BatcherConfig,
	exportFailureMessage string,
	next sender.Sender[request.Request],
) (sender.Sender[request.Request], error) {
	if qCfg.hasBlocking {
		qSet.Telemetry.Logger.Error("using deprecated field `blocking`")
	}
	exportFunc := func(ctx context.Context, req request.Request) error {
		// Have to read the number of items before sending the request since the request can
		// be modified by the downstream components like the batcher.
		itemsCount := req.ItemsCount()
		if errSend := next.Send(ctx, req); errSend != nil {
			qSet.Telemetry.Logger.Error("Exporting failed. Dropping data."+exportFailureMessage,
				zap.Error(errSend), zap.Int("dropped_items", itemsCount))
			return errSend
		}
		return nil
	}

	return queuebatch.NewQueueBatch(qSet, newQueueBatchConfig(qCfg, bCfg), exportFunc)
}

func newQueueBatchConfig(qCfg QueueConfig, bCfg BatcherConfig) queuebatch.Config {
	var qbCfg queuebatch.Config
	// User configured queueing, copy all config.
	if qCfg.Enabled {
		qbCfg = queuebatch.Config{
			Enabled:         true,
			WaitForResult:   qCfg.WaitForResult,
			Sizer:           qCfg.Sizer,
			QueueSize:       qCfg.QueueSize,
			NumConsumers:    qCfg.NumConsumers,
			BlockOnOverflow: qCfg.BlockOnOverflow,
			StorageID:       qCfg.StorageID,
			// TODO: Copy batching configuration as well when available.
		}
		// TODO: Remove this when WithBatcher is removed.
		if bCfg.Enabled {
			qbCfg.Batch = &queuebatch.BatchConfig{
				FlushTimeout: bCfg.FlushTimeout,
				MinSize:      bCfg.MinSize,
				MaxSize:      bCfg.MaxSize,
			}
		}
	} else {
		// This can happen only if the deprecated way to configure batching is used with a "disabled" queue.
		// TODO: Remove this when WithBatcher is removed.
		qbCfg = queuebatch.Config{
			Enabled:         true,
			WaitForResult:   true,
			Sizer:           request.SizerTypeRequests,
			QueueSize:       math.MaxInt,
			NumConsumers:    runtime.NumCPU(),
			BlockOnOverflow: true,
			StorageID:       nil,
			Batch: &queuebatch.BatchConfig{
				FlushTimeout: bCfg.FlushTimeout,
				MinSize:      bCfg.MinSize,
				MaxSize:      bCfg.MaxSize,
			},
		}
	}
	return qbCfg
}

// BatcherConfig defines a configuration for batching requests based on a timeout and a minimum number of items.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type BatcherConfig struct {
	// Enabled indicates whether to not enqueue batches before sending to the consumerSender.
	Enabled bool `mapstructure:"enabled"`

	// FlushTimeout sets the time after which a batch will be sent regardless of its size.
	FlushTimeout time.Duration `mapstructure:"flush_timeout"`

	// SizeConfig sets the size limits for a batch.
	SizeConfig `mapstructure:",squash"`
}

// SizeConfig sets the size limits for a batch.
type SizeConfig struct {
	Sizer request.SizerType `mapstructure:"sizer"`

	// MinSize defines the configuration for the minimum size of a batch.
	MinSize int64 `mapstructure:"min_size"`
	// MaxSize defines the configuration for the maximum size of a batch.
	MaxSize int64 `mapstructure:"max_size"`
}

func (c *BatcherConfig) Validate() error {
	if c.FlushTimeout <= 0 {
		return errors.New("`flush_timeout` must be greater than zero")
	}

	return nil
}

func (c SizeConfig) Validate() error {
	if c.Sizer != request.SizerTypeItems {
		return fmt.Errorf("unsupported sizer type: %q", c.Sizer)
	}
	if c.MinSize < 0 {
		return errors.New("`min_size` must be greater than or equal to zero")
	}
	if c.MaxSize < 0 {
		return errors.New("`max_size` must be greater than or equal to zero")
	}
	if c.MaxSize != 0 && c.MaxSize < c.MinSize {
		return errors.New("`max_size` must be greater than or equal to mix_size")
	}
	return nil
}

func NewDefaultBatcherConfig() BatcherConfig {
	return BatcherConfig{
		Enabled:      true,
		FlushTimeout: 200 * time.Millisecond,
		SizeConfig: SizeConfig{
			Sizer:   request.SizerTypeItems,
			MinSize: 8192,
		},
	}
}
