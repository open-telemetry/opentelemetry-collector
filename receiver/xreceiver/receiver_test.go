// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xreceiver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/internal"
)

var testID = component.MustNewID("test")

func TestNewFactoryWithProfiles(t *testing.T) {
	testType := component.MustNewType("test")
	defaultCfg := struct{}{}
	factory := NewFactory(
		testType,
		func() component.Config { return &defaultCfg },
		WithProfiles(createProfiles, component.StabilityLevelAlpha),
	)
	assert.Equal(t, testType, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())

	assert.Equal(t, component.StabilityLevelAlpha, factory.ProfilesStability())
	_, err := factory.CreateProfiles(context.Background(), receiver.Settings{ID: testID}, &defaultCfg, nil)
	require.NoError(t, err)
	wrongID := component.MustNewID("wrong")
	wrongIDErrStr := internal.ErrIDMismatch(wrongID, testType).Error()
	_, err = factory.CreateProfiles(context.Background(), receiver.Settings{ID: wrongID}, &defaultCfg, nil)
	assert.EqualError(t, err, wrongIDErrStr)
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

func createProfiles(context.Context, receiver.Settings, component.Config, xconsumer.Profiles) (Profiles, error) {
	return nopInstance, nil
}

func TestWithAttributes(t *testing.T) {
	testType := component.MustNewType("test")
	defaultCfg := struct{}{}
	factory := NewFactory(
		testType,
		func() component.Config { return &defaultCfg },
		WithTelemetryAttributes(func(set attribute.Set) attribute.Set {
			return attribute.NewSet(append(set.ToSlice(), attribute.String("foobar", "bar"))...)
		}),
	)
	type factoryWithAttributes interface {
		TelemetryAttributes(attributes attribute.Set) attribute.Set
	}
	result := factory.(factoryWithAttributes).TelemetryAttributes(attribute.NewSet(attribute.String("foo", "bar")))
	assert.Equal(t, attribute.NewSet(attribute.String("foobar", "bar"), attribute.String("foo", "bar")), result)
	// test without attributes set
	factory = NewFactory(
		testType,
		func() component.Config { return &defaultCfg },
	)
	result = factory.(factoryWithAttributes).TelemetryAttributes(attribute.NewSet())
	assert.Empty(t, result.ToSlice())
}
