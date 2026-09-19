// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/cmd/mdatagen/internal"

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"

	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

type SendingQueueSupport string

const (
	SendingQueueSupportDefault      SendingQueueSupport = "default"
	SendingQueueSupportHasOverrides SendingQueueSupport = "has_overrides"
	SendingQueueSupportOmitted      SendingQueueSupport = "omitted"
)

type SendingQueue struct {
	Support   SendingQueueSupport   `mapstructure:"support"`
	Rationale string                `mapstructure:"rationale"`
	Overrides SendingQueueOverrides `mapstructure:"overrides"`
}

type SendingQueueOverrides map[string]any

type parsedSendingQueueOverrides struct {
	enabled        bool
	queue          map[string]any
	batchPresent   bool
	batchEnabled   bool
	batchOverrides map[string]any
}

func (sq *SendingQueue) Validate() error {
	switch sq.Support {
	case SendingQueueSupportDefault:
		if len(sq.Overrides) != 0 {
			return errors.New("sending_queue.overrides requires support to be has_overrides")
		}
	case SendingQueueSupportHasOverrides:
		if len(sq.Overrides) == 0 {
			return errors.New("sending_queue.overrides is required when support is has_overrides")
		}
	case SendingQueueSupportOmitted:
		var errs error
		if strings.TrimSpace(sq.Rationale) == "" {
			errs = errors.Join(errs, errors.New("sending_queue.rationale is required when support is omitted"))
		}
		if len(sq.Overrides) != 0 {
			errs = errors.Join(errs, errors.New("sending_queue.overrides cannot be set when support is omitted"))
		}
		return errs
	default:
		return fmt.Errorf("sending_queue.support must be one of %q, %q, or %q", SendingQueueSupportDefault, SendingQueueSupportHasOverrides, SendingQueueSupportOmitted)
	}

	parsed, err := sq.Overrides.parse()
	if err != nil {
		return err
	}
	if !parsed.enabled && strings.TrimSpace(sq.Rationale) == "" {
		return errors.New("sending_queue.rationale is required when overrides disable the queue")
	}

	cfg, err := sq.Overrides.Apply(exporterhelper.NewDefaultQueueConfig())
	if err != nil {
		return err
	}
	if err := confmap.Validate(cfg); err != nil {
		return fmt.Errorf("invalid sending_queue.overrides: %w", err)
	}
	return nil
}

func (sq *SendingQueue) IsOmitted() bool {
	return sq.Support == SendingQueueSupportOmitted
}

func (sq *SendingQueue) IsEnabled() bool {
	parsed, err := sq.Overrides.parse()
	return err == nil && parsed.enabled
}

func (sq *SendingQueue) HasQueueOverrides() bool {
	parsed, err := sq.Overrides.parse()
	return err == nil && len(parsed.queue) != 0
}

func (sq *SendingQueue) QueueAssignments() ([]string, error) {
	parsed, err := sq.Overrides.parse()
	if err != nil {
		return nil, err
	}
	optionalCfg, err := sq.Overrides.Apply(exporterhelper.NewDefaultQueueConfig())
	if err != nil {
		return nil, err
	}
	cfg := optionalCfg.GetOrInsertDefault()

	assignments := make([]string, 0, len(parsed.queue)+1)
	if _, ok := parsed.queue["wait_for_result"]; ok {
		assignments = append(assignments, fmt.Sprintf("cfg.WaitForResult = %t", cfg.WaitForResult))
	}
	if _, ok := parsed.queue["sizer"]; ok {
		sizer, err := sizerExpression(cfg.Sizer.String())
		if err != nil {
			return nil, err
		}
		assignments = append(assignments, "cfg.Sizer = "+sizer)
	}
	if _, ok := parsed.queue["queue_size"]; ok {
		assignments = append(assignments, fmt.Sprintf("cfg.QueueSize = %d", cfg.QueueSize))
	}
	if _, ok := parsed.queue["block_on_overflow"]; ok {
		assignments = append(assignments, fmt.Sprintf("cfg.BlockOnOverflow = %t", cfg.BlockOnOverflow))
	}
	if _, ok := parsed.queue["storage"]; ok {
		if cfg.StorageID == nil {
			return nil, errors.New("invalid sending_queue.overrides: storage must not be empty")
		}
		constructor := "component.MustNewID"
		args := strconv.Quote(cfg.StorageID.Type().String())
		if cfg.StorageID.Name() != "" {
			constructor = "component.MustNewIDWithName"
			args += ", " + strconv.Quote(cfg.StorageID.Name())
		}
		assignments = append(assignments,
			"storageID := "+constructor+"("+args+")",
			"cfg.StorageID = &storageID",
		)
	}
	if _, ok := parsed.queue["num_consumers"]; ok {
		assignments = append(assignments, fmt.Sprintf("cfg.NumConsumers = %d", cfg.NumConsumers))
	}
	return assignments, nil
}

func (sq *SendingQueue) HasBatchOverrides() bool {
	parsed, err := sq.Overrides.parse()
	return err == nil && parsed.batchPresent
}

func (sq *SendingQueue) IsBatchEnabled() bool {
	parsed, err := sq.Overrides.parse()
	return err == nil && parsed.batchEnabled
}

func (sq *SendingQueue) BatchAssignments() ([]string, error) {
	parsed, err := sq.Overrides.parse()
	if err != nil {
		return nil, err
	}
	if !parsed.batchPresent {
		return nil, nil
	}
	optionalCfg, err := sq.Overrides.Apply(exporterhelper.NewDefaultQueueConfig())
	if err != nil {
		return nil, err
	}
	cfg := optionalCfg.GetOrInsertDefault()
	batchCfg := cfg.Batch.GetOrInsertDefault()

	assignments := make([]string, 0, len(parsed.batchOverrides))
	if _, ok := parsed.batchOverrides["flush_timeout"]; ok {
		assignments = append(assignments, fmt.Sprintf("batchCfg.FlushTimeout = time.Duration(%d)", batchCfg.FlushTimeout))
	}
	if _, ok := parsed.batchOverrides["sizer"]; ok {
		sizer, err := sizerExpression(batchCfg.Sizer.String())
		if err != nil {
			return nil, err
		}
		assignments = append(assignments, "batchCfg.Sizer = "+sizer)
	}
	if _, ok := parsed.batchOverrides["min_size"]; ok {
		assignments = append(assignments, fmt.Sprintf("batchCfg.MinSize = %d", batchCfg.MinSize))
	}
	if _, ok := parsed.batchOverrides["max_size"]; ok {
		assignments = append(assignments, fmt.Sprintf("batchCfg.MaxSize = %d", batchCfg.MaxSize))
	}
	if _, ok := parsed.batchOverrides["partition"]; ok {
		keys := make([]string, 0, len(batchCfg.Partition.MetadataKeys))
		for _, key := range batchCfg.Partition.MetadataKeys {
			keys = append(keys, strconv.Quote(key))
		}
		assignments = append(assignments, "batchCfg.Partition.MetadataKeys = []string{"+strings.Join(keys, ", ")+"}")
	}
	return assignments, nil
}

func (sq *SendingQueue) HasStorageOverride() bool {
	parsed, err := sq.Overrides.parse()
	if err != nil {
		return false
	}
	_, ok := parsed.queue["storage"]
	return ok
}

func (sq *SendingQueue) HasBatchFlushTimeoutOverride() bool {
	parsed, err := sq.Overrides.parse()
	if err != nil {
		return false
	}
	_, ok := parsed.batchOverrides["flush_timeout"]
	return ok
}

func (sq *SendingQueue) YAMLQueueOverrides() (string, error) {
	parsed, err := sq.Overrides.parse()
	if err != nil {
		return "", err
	}
	overrides := make(map[string]any, len(parsed.queue)+1)
	for key, value := range parsed.queue {
		overrides[key] = value
	}
	if parsed.batchPresent {
		batch := make(map[string]any, len(parsed.batchOverrides)+1)
		for key, value := range parsed.batchOverrides {
			batch[key] = value
		}
		batch["enabled"] = parsed.batchEnabled
		overrides["batch"] = batch
	}
	if len(overrides) == 0 {
		return "", nil
	}
	encoded, err := yaml.Marshal(overrides)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(encoded)), nil
}

func (sq *SendingQueue) IndentedYAMLQueueOverrides() (string, error) {
	overrides, err := sq.YAMLQueueOverrides()
	if err != nil || overrides == "" {
		return overrides, err
	}
	return "  " + strings.ReplaceAll(overrides, "\n", "\n  "), nil
}

func (overrides SendingQueueOverrides) Apply(cfg exporterhelper.QueueBatchConfig) (configoptional.Optional[exporterhelper.QueueBatchConfig], error) {
	parsed, err := overrides.parse()
	if err != nil {
		return configoptional.None[exporterhelper.QueueBatchConfig](), err
	}
	if err := confmap.NewFromStringMap(parsed.queue).Unmarshal(&cfg); err != nil {
		return configoptional.None[exporterhelper.QueueBatchConfig](), fmt.Errorf("invalid sending_queue.overrides: %w", err)
	}
	if parsed.batchPresent {
		batchCfg := *cfg.Batch.GetOrInsertDefault()
		if err := confmap.NewFromStringMap(parsed.batchOverrides).Unmarshal(&batchCfg); err != nil {
			return configoptional.None[exporterhelper.QueueBatchConfig](), fmt.Errorf("invalid sending_queue.overrides.batch: %w", err)
		}
		if parsed.batchEnabled {
			cfg.Batch = configoptional.Some(batchCfg)
		} else {
			cfg.Batch = configoptional.Default(batchCfg)
		}
	}
	if parsed.enabled {
		return configoptional.Some(cfg), nil
	}
	return configoptional.Default(cfg), nil
}

func (overrides SendingQueueOverrides) parse() (parsedSendingQueueOverrides, error) {
	parsed := parsedSendingQueueOverrides{
		enabled: true,
		queue:   make(map[string]any, len(overrides)),
	}
	for key, value := range overrides {
		switch key {
		case "enabled":
			enabled, ok := value.(bool)
			if !ok {
				return parsedSendingQueueOverrides{}, fmt.Errorf("invalid sending_queue.overrides: enabled must be a boolean, got %T", value)
			}
			parsed.enabled = enabled
		case "batch":
			batchOverrides, ok := value.(map[string]any)
			if !ok {
				return parsedSendingQueueOverrides{}, fmt.Errorf("invalid sending_queue.overrides: batch must be a map, got %T", value)
			}
			parsed.batchPresent = true
			parsed.batchEnabled = true
			parsed.batchOverrides = make(map[string]any, len(batchOverrides))
			for batchKey, batchValue := range batchOverrides {
				if batchKey != "enabled" {
					parsed.batchOverrides[batchKey] = batchValue
					continue
				}
				batchEnabled, ok := batchValue.(bool)
				if !ok {
					return parsedSendingQueueOverrides{}, fmt.Errorf("invalid sending_queue.overrides.batch: enabled must be a boolean, got %T", batchValue)
				}
				parsed.batchEnabled = batchEnabled
			}
		default:
			parsed.queue[key] = value
		}
	}
	return parsed, nil
}

func sizerExpression(value string) (string, error) {
	switch value {
	case "bytes":
		return "exporterhelper.RequestSizerTypeBytes", nil
	case "items":
		return "exporterhelper.RequestSizerTypeItems", nil
	case "requests":
		return "exporterhelper.RequestSizerTypeRequests", nil
	default:
		return "", fmt.Errorf("invalid sending_queue sizer %q", value)
	}
}
