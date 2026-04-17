// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configunmarshaler // import "go.opentelemetry.io/collector/otelcol/internal/configunmarshaler"

import (
	"errors"
	"fmt"

	otelconf "go.opentelemetry.io/contrib/otelconf/x"
	"golang.org/x/exp/maps"
	"gopkg.in/yaml.v3"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
)

type Configs[F component.Factory] struct {
	cfgs     map[component.ID]component.Config
	otelCfgs map[component.ID]otelconf.OpenTelemetryConfiguration

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
	c.otelCfgs = make(map[component.ID]otelconf.OpenTelemetryConfiguration)
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

		// Extract per-component OTel SDK configuration before component unmarshalling.
		// The "telemetry" key follows the OpenTelemetry Configuration schema and can
		// configure a LoggerProvider (and in the future MeterProvider/TracerProvider)
		// per component. The key is stripped so the component's unmarshaller doesn't
		// fail on unknown fields.
		if sub.IsSet("telemetry") {
			telSub, err := sub.Sub("telemetry")
			if err != nil {
				return errorUnmarshalError(id, err)
			}
			// Use otelconf.ParseYAML instead of confmap.Unmarshal because
			// the otelconf generated types use `interface{}` with ",remain"
			// tags that are incompatible with confmap's mapstructure decoder.
			telMap := telSub.ToStringMap()
			if _, ok := telMap["file_format"]; !ok {
				telMap["file_format"] = "0.3"
			}
			yamlBytes, err := yaml.Marshal(telMap)
			if err != nil {
				return errorUnmarshalError(id, fmt.Errorf("invalid telemetry config: %w", err))
			}
			otelCfg, err := otelconf.ParseYAML(yamlBytes)
			if err != nil {
				return errorUnmarshalError(id, fmt.Errorf("invalid telemetry config: %w", err))
			}
			c.otelCfgs[id] = *otelCfg
			sub.Delete("telemetry")
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

// OtelConfigs returns per-component OpenTelemetry SDK configurations extracted
// from the "telemetry" key in each component's configuration section.
func (c *Configs[F]) OtelConfigs() map[component.ID]otelconf.OpenTelemetryConfiguration {
	return c.otelCfgs
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
