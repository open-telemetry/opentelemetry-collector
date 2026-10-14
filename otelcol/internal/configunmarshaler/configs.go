// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configunmarshaler // import "go.opentelemetry.io/collector/otelcol/internal/configunmarshaler"

import (
	"errors"
	"fmt"

	"go.uber.org/zap/zapcore"
	"golang.org/x/exp/maps"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
)

// TelemetryOverride holds per-component telemetry configuration extracted from
// the component's config map before delegating to the component factory.
// This allows transparent infrastructure-level features (e.g. per-component
// log level) without requiring any changes to component factories.
type TelemetryOverride struct {
	// LogLevel, when non-nil, overrides the service-level default log level
	// for this component's own logger. It does not affect child components.
	LogLevel *zapcore.Level
}

type Configs[F component.Factory] struct {
	cfgs map[component.ID]component.Config

	// telemetryOverrides stores per-component telemetry configuration overrides
	// extracted from component configs during Unmarshal.
	telemetryOverrides map[component.ID]TelemetryOverride

	factories map[component.Type]F
}

func NewConfigs[F component.Factory](factories map[component.Type]F) *Configs[F] {
	return &Configs[F]{factories: factories}
}

func (c *Configs[F]) Unmarshal(conf *confmap.Conf) error {
	rawCfgs := make(map[component.ID]map[string]any)
	if err := conf.Unmarshal(&rawCfgs); err != nil {
		return err
	}

	// Prepare resulting map.
	c.cfgs = make(map[component.ID]component.Config)
	c.telemetryOverrides = make(map[component.ID]TelemetryOverride)

	// Iterate over raw configs and create a config for each.
	for id := range rawCfgs {
		// Find factory based on component kind and type that we read from config source.
		factory, ok := c.factories[id.Type()]
		if !ok {
			return errorUnknownType(id, maps.Keys(c.factories))
		}

		// Get the configuration from the confmap.Conf to preserve internal representation.
		sub, err := conf.Sub(id.String())
		if err != nil {
			return errorUnmarshalError(id, err)
		}

		// Extract per-component telemetry override BEFORE delegating to the factory.
		// Component factories have no knowledge of this infrastructure-level feature.
		if override, ok := extractTelemetryOverride(sub); ok {
			c.telemetryOverrides[id] = override
		}

		// Create the default config for this component.
		cfg := factory.CreateDefaultConfig()

		// Now that the default config struct is created we can Unmarshal into it,
		// and it will apply user-defined config on top of the default.
		if err := sub.Unmarshal(&cfg); err != nil {
			return errorUnmarshalError(id, err)
		}

		c.cfgs[id] = cfg
	}

	return nil
}

func (c *Configs[F]) Configs() map[component.ID]component.Config {
	return c.cfgs
}

// TelemetryOverrides returns the per-component telemetry overrides collected
// during Unmarshal. The returned map is keyed by component ID.
func (c *Configs[F]) TelemetryOverrides() map[component.ID]TelemetryOverride {
	return c.telemetryOverrides
}

// extractTelemetryOverride checks if the given confmap contains a "telemetry"
// key, and if so extracts the "logs.level" value into a TelemetryOverride.
// It also strips the "telemetry" key from the confmap so it won't be passed
// to the component factory's config struct.
func extractTelemetryOverride(sub *confmap.Conf) (TelemetryOverride, bool) {
	if !sub.IsSet("telemetry") {
		return TelemetryOverride{}, false
	}

	var override TelemetryOverride

	// Try to get telemetry.logs.level
	if sub.IsSet("telemetry::logs::level") {
		levelStr, ok := sub.Get("telemetry::logs::level").(string)
		if ok {
			var level zapcore.Level
			if err := level.UnmarshalText([]byte(levelStr)); err == nil {
				override.LogLevel = &level
			}
		}
	}

	// Strip the telemetry key so it won't be passed to the component factory
	sub.Delete("telemetry")

	return override, true
}

func errorUnknownType(id component.ID, factories []component.Type) error {
	if id.Type().String() == "logging" {
		return errors.New("the logging exporter has been deprecated, use the debug exporter instead")
	}
	return fmt.Errorf("unknown type: %q for id: %q (valid values: %v)", id.Type(), id, factories)
}

func errorUnmarshalError(id component.ID, err error) error {
	return fmt.Errorf("error reading configuration for %q: %w", id, err)
}
