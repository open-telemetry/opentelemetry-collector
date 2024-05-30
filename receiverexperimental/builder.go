// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package receiverexperimental // import "go.opentelemetry.io/collector/receiverexperimental"

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component" // Builder receiver is a helper struct that given a set of Configs and Factories helps with creating receivers.
	"go.opentelemetry.io/collector/consumerexperimental"
	"go.opentelemetry.io/collector/receiver"
)

var (
	errNilNextConsumer = errors.New("nil next Consumer")
)

type Builder struct {
	cfgs      map[component.ID]component.Config
	factories map[component.Type]Factory
}

// NewBuilder creates a new receiver.Builder to help with creating components form a set of configs and factories.
func NewBuilder(cfgs map[component.ID]component.Config, factories map[component.Type]Factory) *Builder {
	return &Builder{cfgs: cfgs, factories: factories}
}

// CreateProfiles creates a Profiles receiver based on the settings and config.
func (b *Builder) CreateProfiles(ctx context.Context, set receiver.CreateSettings, next consumerexperimental.Profiles) (Profiles, error) {
	if next == nil {
		return nil, errNilNextConsumer
	}
	cfg, existsCfg := b.cfgs[set.ID]
	if !existsCfg {
		return nil, fmt.Errorf("receiver %q is not configured", set.ID)
	}

	f, existsFactory := b.factories[set.ID.Type()]
	if !existsFactory {
		return nil, fmt.Errorf("receiver factory not available for: %q", set.ID)
	}

	logStabilityLevel(set.Logger, f.ProfilesReceiverStability())
	return f.CreateProfilesReceiver(ctx, set, cfg, next)
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
