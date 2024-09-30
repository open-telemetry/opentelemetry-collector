// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package service // import "go.opentelemetry.io/collector/service"

import (
	"fmt"

	"go.opentelemetry.io/collector/service/extensions"
	"go.opentelemetry.io/collector/service/pipelines"
	"go.opentelemetry.io/collector/service/telemetry"
)

// Config defines the configurable components of the Service.
type Config struct {
	// Telemetry is the configuration for collector's own telemetry.
	Telemetry telemetry.Config `mapstructure:"telemetry"`

	// Extensions are the ordered list of extensions configured for the service.
	Extensions extensions.Config `mapstructure:"extensions"`

	// Pipelines are the set of data pipelines configured for the service.
	Pipelines pipelines.Config `mapstructure:"pipelines"`

	// Pipelines are the set of data pipelines configured for the service.
	//
	// Deprecated: [v0.111.0] Use Pipelines instead.
	PipelinesWithPipelineID pipelines.ConfigWithPipelineID `mapstructure:"-"` // nolint
}

func (cfg *Config) Validate() error {
	if len(cfg.Pipelines) > 0 && len(cfg.PipelinesWithPipelineID) > 0 {
		return fmt.Errorf("service::pipelines config validation failed: cannot configure both Pipelines and PipelinesWithPipelineID")
	}

	if len(cfg.PipelinesWithPipelineID) > 0 {
		if err := cfg.PipelinesWithPipelineID.Validate(); err != nil {
			return fmt.Errorf("service::pipelines config validation failed: %w", err)
		}
	} else {
		if err := cfg.Pipelines.Validate(); err != nil {
			return fmt.Errorf("service::pipelines config validation failed: %w", err)
		}
	}

	if err := cfg.Telemetry.Validate(); err != nil {
		fmt.Printf("service::telemetry config validation failed: %v\n", err)
	}

	return nil
}
