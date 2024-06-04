// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package processorexperimental // import "go.opentelemetry.io/collector/processorexperimental"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumerexperimental"
	"go.opentelemetry.io/collector/processor" // Builder processor is a helper struct that given a set of Configs and Factories helps with creating processors.
	"go.uber.org/zap"
)

type Builder struct {
	cfgs      map[component.ID]component.Config
	factories map[component.Type]Factory
}

// NewBuilder creates a new processor.Builder to help with creating components form a set of configs and factories.
func NewBuilder(cfgs map[component.ID]component.Config, factories map[component.Type]Factory) *Builder {
	return &Builder{cfgs: cfgs, factories: factories}
}

// CreateProfiles creates a Profiles processor based on the settings and config.
func (b *Builder) CreateProfiles(ctx context.Context, set processor.CreateSettings, next consumerexperimental.Profiles) (Profiles, error) {
	if next == nil {
		return nil, errNilNextConsumer
	}
	cfg, existsCfg := b.cfgs[set.ID]
	if !existsCfg {
		return nil, fmt.Errorf("processor %q is not configured", set.ID)
	}

	f, existsFactory := b.factories[set.ID.Type()]
	if !existsFactory {
		return nil, fmt.Errorf("processor factory not available for: %q", set.ID)
	}

	logStabilityLevel(set.Logger, f.ProfilesProcessorStability())
	return f.CreateProfilesProcessor(ctx, set, cfg, next)
}

func (b *Builder) Factory(componentType component.Type) component.Factory {
	return b.factories[componentType]
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
