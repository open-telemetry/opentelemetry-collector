// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterexperimental // import "go.opentelemetry.io/collector/exporterexperimental"

import (
	"context" // Builder exporter is a helper struct that given a set of Configs and Factories helps with creating exporters.
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.uber.org/zap"
)

type Builder struct {
	cfgs      map[component.ID]component.Config
	factories map[component.Type]Factory
}

// NewBuilder creates a new exporter.Builder to help with creating components form a set of configs and factories.
func NewBuilder(cfgs map[component.ID]component.Config, factories map[component.Type]Factory) *Builder {
	return &Builder{cfgs: cfgs, factories: factories}
}

// CreateProfiless creates a Profiless exporter based on the settings and config.
func (b *Builder) CreateProfiles(ctx context.Context, set exporter.Settings) (Profiles, error) {
	cfg, existsCfg := b.cfgs[set.ID]
	if !existsCfg {
		return nil, fmt.Errorf("exporter %q is not configured", set.ID)
	}

	f, existsFactory := b.factories[set.ID.Type()]
	if !existsFactory {
		return nil, fmt.Errorf("exporter factory not available for: %q", set.ID)
	}

	logStabilityLevel(set.Logger, f.ProfilesExporterStability())
	return f.CreateProfilesExporter(ctx, set, cfg)
}

// logStabilityLevel logs the stability level of a component. The log level is set to info for
// undefined, unmaintained, deprecated and development. The log level is set to debug
// for alpha, beta and stable.
func logStabilityLevel(logger *zap.Logger, sl component.StabilityLevel) {
	if sl >= component.StabilityLevelAlpha {
		logger.Debug(sl.LogMessage())
	} else {
		logger.Info(sl.LogMessage())
	}
}
