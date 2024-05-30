// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package receiverexperimental // import "go.opentelemetry.io/collector/receiverexperimental"

import (
	"context"

	"go.opentelemetry.io/collector/component" // Profiles receiver receives profiles.
	"go.opentelemetry.io/collector/consumerexperimental"
	"go.opentelemetry.io/collector/receiver"
)

// Its purpose is to translate data from any format to the collector's internal trace format.
// ProfilesReceiver feeds a consumerexperimental.Profiles with data.
//
// For example, it could be the OTLP data source which translates OTLP spans into pprofile.Profiles.
type Profiles interface {
	component.Component
}

// CreateProflesFunc is the equivalent of Factory.CreateProfiles.
type CreateProfilesFunc func(context.Context, receiver.CreateSettings, component.Config, consumerexperimental.Profiles) (Profiles, error)

// CreateProfilesReceiver implements Factory.CreateProfilesReceiver().
func (f CreateProfilesFunc) CreateProfilesReceiver(
	ctx context.Context,
	set receiver.CreateSettings,
	cfg component.Config,
	nextConsumer consumerexperimental.Profiles) (Profiles, error) {
	if f == nil {
		return nil, component.ErrDataTypeIsNotSupported
	}
	return f(ctx, set, cfg, nextConsumer)
}
