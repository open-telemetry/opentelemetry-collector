// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/cmd/mdatagen/internal"

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"

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
	Support   SendingQueueSupport `mapstructure:"support"`
	Rationale string              `mapstructure:"rationale"`
	Overrides map[string]any      `mapstructure:"overrides"`
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

	enabled, queueOverrides, err := sq.splitOverrides()
	if err != nil {
		return err
	}
	if !enabled && strings.TrimSpace(sq.Rationale) == "" {
		return errors.New("sending_queue.rationale is required when overrides disable the queue")
	}

	cfg := exporterhelper.NewDefaultQueueConfig()
	if err := confmap.NewFromStringMap(queueOverrides).Unmarshal(&cfg); err != nil {
		return fmt.Errorf("invalid sending_queue.overrides: %w", err)
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
	enabled, _, err := sq.splitOverrides()
	return err == nil && enabled
}

func (sq *SendingQueue) HasQueueOverrides() bool {
	_, overrides, err := sq.splitOverrides()
	return err == nil && len(overrides) != 0
}

func (sq *SendingQueue) GoQueueOverrides() (string, error) {
	_, overrides, err := sq.splitOverrides()
	if err != nil {
		return "", err
	}
	return renderGoValue(overrides)
}

func (sq *SendingQueue) YAMLQueueOverrides() (string, error) {
	_, overrides, err := sq.splitOverrides()
	if err != nil || len(overrides) == 0 {
		return "", err
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

func (sq *SendingQueue) splitOverrides() (bool, map[string]any, error) {
	enabled := true
	overrides := make(map[string]any, len(sq.Overrides))
	for key, value := range sq.Overrides {
		if key != "enabled" {
			overrides[key] = value
			continue
		}
		var ok bool
		enabled, ok = value.(bool)
		if !ok {
			return false, nil, fmt.Errorf("invalid sending_queue.overrides: enabled must be a boolean, got %T", value)
		}
	}
	return enabled, overrides, nil
}

func renderGoValue(value any) (string, error) {
	switch value := value.(type) {
	case nil:
		return "nil", nil
	case bool:
		return strconv.FormatBool(value), nil
	case string:
		return strconv.Quote(value), nil
	case int:
		return strconv.Itoa(value), nil
	case int8:
		return strconv.FormatInt(int64(value), 10), nil
	case int16:
		return strconv.FormatInt(int64(value), 10), nil
	case int32:
		return strconv.FormatInt(int64(value), 10), nil
	case int64:
		return strconv.FormatInt(value, 10), nil
	case uint:
		return strconv.FormatUint(uint64(value), 10), nil
	case uint8:
		return strconv.FormatUint(uint64(value), 10), nil
	case uint16:
		return strconv.FormatUint(uint64(value), 10), nil
	case uint32:
		return strconv.FormatUint(uint64(value), 10), nil
	case uint64:
		return strconv.FormatUint(value, 10), nil
	case float32:
		return strconv.FormatFloat(float64(value), 'g', -1, 32), nil
	case float64:
		return strconv.FormatFloat(value, 'g', -1, 64), nil
	case []any:
		values := make([]string, 0, len(value))
		for _, item := range value {
			rendered, err := renderGoValue(item)
			if err != nil {
				return "", err
			}
			values = append(values, rendered)
		}
		return "[]any{" + strings.Join(values, ", ") + "}", nil
	case map[string]any:
		keys := make([]string, 0, len(value))
		for key := range value {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		values := make([]string, 0, len(value))
		for _, key := range keys {
			rendered, err := renderGoValue(value[key])
			if err != nil {
				return "", err
			}
			values = append(values, strconv.Quote(key)+": "+rendered)
		}
		return "map[string]any{" + strings.Join(values, ", ") + "}", nil
	default:
		return "", fmt.Errorf("unsupported sending_queue override value type %T", value)
	}
}
