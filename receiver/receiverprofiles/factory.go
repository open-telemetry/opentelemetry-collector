// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package receiverprofiles // import "go.opentelemetry.io/collector/receiver/receiverprofiles"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumerprofiles"
	"go.opentelemetry.io/collector/receiver"
)

// Factory is a factory interface for receivers that implement profiles.
//
// This interface cannot be directly implemented. Implementations must
// use the NewReceiverFactory to implement it.
type Factory interface {
	component.Factory

	// CreateProfilesReceiver creates a ProfilesReceiver based on this config.
	// If the receiver type does not support tracing or if the config is not valid
	// an error will be returned instead. `nextConsumer` is never nil.
	CreateProfilesReceiver(ctx context.Context, set receiver.Settings, cfg component.Config, nextConsumer consumerprofiles.Profiles) (Profiles, error)

	// ProfilesReceiverStability gets the stability level of the ProfilesReceiver.
	ProfilesReceiverStability() component.StabilityLevel
}
