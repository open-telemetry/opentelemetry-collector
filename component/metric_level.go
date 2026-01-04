// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package component // import "go.opentelemetry.io/collector/component"

import "sync"

// MetricLevelConfig declares the minimum service telemetry level required for
// a given meter or instrument. Components can register these declarations so
// the service can derive default views that drop metrics when the level is
// below the configured threshold.
type MetricLevelConfig struct {
	// MeterName is the fully-qualified meter name emitting the metric.
	MeterName string

	// InstrumentName optionally scopes the config to a specific instrument. When
	// empty, the config applies to the complete meter.
	InstrumentName string

	// Level is the minimum service telemetry level required for the meter or
	// instrument to be enabled.
	Level MetricLevel
}

// MetricLevel mirrors service::telemetry::metrics::level values.
type MetricLevel int32

// metricLevelRegistry is a global registry for metric level configurations.
var metricLevelRegistry = struct {
	sync.RWMutex
	configs []MetricLevelConfig
	byMeter map[string][]int
}{byMeter: make(map[string][]int)}

// RegisterMetricLevelConfigs registers one or more MetricLevelConfig entries.
func RegisterMetricLevelConfigs(configs ...MetricLevelConfig) {
	if len(configs) == 0 {
		return
	}

	metricLevelRegistry.Lock()
	defer metricLevelRegistry.Unlock()

	for _, cfg := range configs {
		if cfg.MeterName == "" {
			panic("component: MetricLevelConfig requires MeterName")
		}
		idx := len(metricLevelRegistry.configs)
		metricLevelRegistry.configs = append(metricLevelRegistry.configs, cfg)
		metricLevelRegistry.byMeter[cfg.MeterName] = append(metricLevelRegistry.byMeter[cfg.MeterName], idx)
	}
}

// RegisteredMetricLevelConfigs returns all registered metric level declarations.
// The returned slice is a copy to prevent external modification of the registry.
// For better performance when iterating, consider using RegisteredMetricLevelConfigsByMeter
// if you only need configs for specific meters.
func RegisteredMetricLevelConfigs() []MetricLevelConfig {
	metricLevelRegistry.RLock()
	defer metricLevelRegistry.RUnlock()

	out := make([]MetricLevelConfig, len(metricLevelRegistry.configs))
	copy(out, metricLevelRegistry.configs)
	return out
}

// RegisteredMetricLevelConfigsByMeter returns all registered metric level declarations
// for the given meter name.
func RegisteredMetricLevelConfigsByMeter(meterName string) []MetricLevelConfig {
	metricLevelRegistry.RLock()
	defer metricLevelRegistry.RUnlock()

	indices, exists := metricLevelRegistry.byMeter[meterName]
	if !exists {
		return nil
	}

	configs := make([]MetricLevelConfig, 0, len(indices))
	for _, idx := range indices {
		configs = append(configs, metricLevelRegistry.configs[idx])
	}
	return configs
}

// ResetMetricLevelRegistryForTesting resets the global registry for testing.
// This function is exported for use in test files (including tests in other packages)
// and should never be called from production code.
func ResetMetricLevelRegistryForTesting() {
	metricLevelRegistry.Lock()
	defer metricLevelRegistry.Unlock()
	metricLevelRegistry.configs = nil
	metricLevelRegistry.byMeter = make(map[string][]int)
}
