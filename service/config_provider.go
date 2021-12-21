// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package service // import "go.opentelemetry.io/collector/service"

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configmapprovider"
	"go.opentelemetry.io/collector/config/configunmarshaler"
	"go.opentelemetry.io/collector/config/experimental/configsource"
	"go.opentelemetry.io/collector/internal/configprovider"
)

// ConfigProvider provides the service configuration.
//
// The typical usage is the following:
//
//		cfgProvider.Get(...)
//		cfgProvider.Watch() // wait for an event.
//		cfgProvider.Get(...)
//		cfgProvider.Watch() // wait for an event.
//		// repeat Get/Watch cycle until it is time to shut down the Collector process.
//		cfgProvider.Shutdown()
type ConfigProvider interface {
	// Get returns the service configuration, or error otherwise.
	//
	// Should never be called concurrently with itself, Watch or Shutdown.
	Get(ctx context.Context, factories component.Factories) (*config.Config, error)

	// Watch blocks until any configuration change was detected or an unrecoverable error
	// happened during monitoring the configuration changes.
	//
	// Error is nil if the configuration is changed and needs to be re-fetched. Any non-nil
	// error indicates that there was a problem with watching the config changes.
	//
	// Should never be called concurrently with itself or Get.
	Watch() <-chan error

	// Shutdown signals that the provider is no longer in use and the that should close
	// and release any resources that it may have created.
	//
	// This function must terminate the Watch channel.
	//
	// Should never be called concurrently with itself or Get.
	Shutdown(ctx context.Context) error
}

type configProvider struct {
	configMapProvider configmapprovider.Provider
	configUnmarshaler configunmarshaler.ConfigUnmarshaler

	sync.Mutex
	ret     configmapprovider.Retrieved
	watcher chan error
}

// newConfigProvider returns a new ConfigProvider that provides the configuration using the given
// `configMapProvider` and the given `configUnmarshaler`.
func newConfigProvider(configMapProvider configmapprovider.Provider, configUnmarshaler configunmarshaler.ConfigUnmarshaler) ConfigProvider {
	return &configProvider{
		configMapProvider: configMapProvider,
		configUnmarshaler: configUnmarshaler,
		watcher:           make(chan error, 1),
	}
}

// NewDefaultConfigProvider returns the default ConfigProvider, and it creates configuration from a file
// defined by the given configFile and overwrites fields using properties.
func NewDefaultConfigProvider(configFileName string, properties []string) ConfigProvider {
	return &configProvider{
		configMapProvider: configprovider.NewDefaultMapProvider(configFileName, properties),
		configUnmarshaler: configunmarshaler.NewDefault(),
		watcher:           make(chan error, 1),
	}
}

func (cm *configProvider) Get(ctx context.Context, factories component.Factories) (*config.Config, error) {
	// First check if already an active watching, close that if any.
	if err := cm.closeIfNeeded(ctx); err != nil {
		return nil, fmt.Errorf("cannot close previous watch: %w", err)
	}

	var err error
	cm.ret, err = cm.configMapProvider.Retrieve(ctx, cm.onChange)
	if err != nil {
		// Nothing to close, no valid retrieved value.
		cm.ret = nil
		return nil, fmt.Errorf("cannot retrieve the configuration: %w", err)
	}

	var cfg *config.Config
	m, err := cm.ret.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot get the configuration: %w", err)
	}
	if cfg, err = cm.configUnmarshaler.Unmarshal(m, factories); err != nil {
		return nil, fmt.Errorf("cannot unmarshal the configuration: %w", err)
	}

	if err = cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

func (cm *configProvider) Watch() <-chan error {
	return cm.watcher
}

func (cm *configProvider) onChange(event *configmapprovider.ChangeEvent) {
	// TODO: Remove check for configsource.ErrSessionClosed when providers updated to not call onChange when closed.
	if event.Error != configsource.ErrSessionClosed {
		cm.watcher <- event.Error
	}
}

func (cm *configProvider) closeIfNeeded(ctx context.Context) error {
	if cm.ret != nil {
		return cm.ret.Close(ctx)
	}
	return nil
}

func (cm *configProvider) Shutdown(ctx context.Context) error {
	close(cm.watcher)
	return multierr.Combine(cm.closeIfNeeded(ctx), cm.configMapProvider.Shutdown(ctx))
}
