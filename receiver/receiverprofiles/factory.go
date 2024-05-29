package receiverprofiles

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/internal"
)

type Profiles = internal.Profiles

// Factory is factory interface for receivers that can create profiles receivers.
// It is an optional interface that receivers can implement if they support profiles.
type Factory interface {
	CreateProfilesReceiver(ctx context.Context, set receiver.CreateSettings, cfg component.Config, nextConsumer consumer.Profiles) (Profiles, error)
	internal.Internal
}

type CreateProfilesFunc = internal.CreateProfilesFunc

// WithProfiles adds support for profiling to a receiver factory.
func WithProfiles(createProfilesReceiver CreateProfilesFunc, sl component.StabilityLevel) receiver.FactoryOption {
	return internal.WithProfiles(createProfilesReceiver, sl)
}
