// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configunmarshaler // import "go.opentelemetry.io/collector/otelcol/internal/configunmarshaler"

import (
	"fmt"

	"go.uber.org/zap"
	"golang.org/x/exp/maps"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/internal/strictlytypedgate"
)

type Configs[F component.Factory] struct {
	cfgs      map[component.ID]component.Config
	logger    *zap.Logger
	factories map[component.Type]F
}

func NewConfigs[F component.Factory](logger *zap.Logger, factories map[component.Type]F) *Configs[F] {
	return &Configs[F]{factories: factories}
}

func (c *Configs[F]) Unmarshal(conf *confmap.Conf) error {
	var opts []confmap.UnmarshalOption
	// If the feature gate is enabled, we will require strictly typed input.
	if strictlytypedgate.StrictlyTypedInputGate.IsEnabled() {
		opts = append(opts, confmap.WithStrictlyTypedInput())
	}

	rawCfgs := make(map[component.ID]map[string]any)
	if err := conf.Unmarshal(&rawCfgs, opts...); err != nil {
		return err
	}

	// Prepare resulting map.
	c.cfgs = make(map[component.ID]component.Config)
	// Iterate over raw configs and create a config for each.
	for id, value := range rawCfgs {
		// Find factory based on component kind and type that we read from config source.
		factory, ok := c.factories[id.Type()]
		if !ok {
			return errorUnknownType(id, maps.Keys(c.factories))
		}

		// Create the default config for this component.
		cfg := factory.CreateDefaultConfig()

		// Now that the default config struct is created we can Unmarshal into it,
		// and it will apply user-defined config on top of the default.
		conf := confmap.NewFromStringMap(value)
		if err := conf.Unmarshal(&cfg, opts...); err != nil {
			return errorUnmarshalError(id, err)
		}

		opts = append(opts, confmap.WithStrictlyTypedInput())
		confCopy := conf.Copy()
		cfgCopy := factory.CreateDefaultConfig()
		if err := confCopy.Unmarshal(&cfgCopy, opts...); err != nil {
			c.logger.Warn("Configuration will fail to resolve when we require strictly typed input. Enable the feature gate to check your configuration types.",
				zap.Error(err),
				zap.Stringer("component id", id),
				zap.String("feature gate", "confmap.strictlyTypedInput"),
			)
		}

		c.cfgs[id] = cfg
	}

	return nil
}

func (c *Configs[F]) Configs() map[component.ID]component.Config {
	return c.cfgs
}

func errorUnknownType(id component.ID, factories []component.Type) error {
	return fmt.Errorf("unknown type: %q for id: %q (valid values: %v)", id.Type(), id, factories)
}

func errorUnmarshalError(id component.ID, err error) error {
	return fmt.Errorf("error reading configuration for %q: %w", id, err)
}
