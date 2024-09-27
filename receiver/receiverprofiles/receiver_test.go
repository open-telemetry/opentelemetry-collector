// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package receiverprofiles

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumerprofiles"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver"
)

func TestNewFactoryWithProfiles(t *testing.T) {
	var testType = component.MustNewType("test")
	defaultCfg := struct{}{}
	factory := NewFactory(
		testType,
		func() component.Config { return &defaultCfg },
		WithProfiles(createProfiles, component.StabilityLevelAlpha),
	)
	assert.EqualValues(t, testType, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())

	assert.Equal(t, component.StabilityLevelAlpha, factory.ProfilesReceiverStability())
	_, err := factory.CreateProfilesReceiver(context.Background(), receiver.Settings{}, &defaultCfg, nil)
	assert.NoError(t, err)
}

var nopInstance = &nopReceiver{
	Consumer: consumertest.NewNop(),
}

// nopReceiver stores consumed traces and metrics for testing purposes.
type nopReceiver struct {
	component.StartFunc
	component.ShutdownFunc
	consumertest.Consumer
}

func createProfiles(context.Context, receiver.Settings, component.Config, consumerprofiles.Profiles) (Profiles, error) {
	return nopInstance, nil
}
