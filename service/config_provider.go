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
	"go.opentelemetry.io/collector/service/internal/configprovider"
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
	configMapProviders []configmapprovider.Provider
	cfgMapConverters   []ConfigMapConverterFunc
	configUnmarshaler  configunmarshaler.ConfigUnmarshaler

	sync.Mutex
	ret     configmapprovider.Retrieved
	watcher chan error
}

// NewConfigProvider returns a new ConfigProvider that provides the configuration:
// * Retrieve the config.Map by merging all retrieved maps from all the configmapprovider.Provider in order.
// * Then applies all the ConfigMapConverterFunc in the given order.
// * Then unmarshalls the final config.Config using the given configunmarshaler.ConfigUnmarshaler.
//
// Notice: This API is experimental.
func NewConfigProvider(configMapProviders []configmapprovider.Provider, cfgMapConverters []ConfigMapConverterFunc, configUnmarshaler configunmarshaler.ConfigUnmarshaler) ConfigProvider {
	return &configProvider{
		configMapProviders: configMapProviders,
		cfgMapConverters:   cfgMapConverters,
		configUnmarshaler:  configUnmarshaler,
		watcher:            make(chan error, 1),
	}
}

// ConfigMapConverterFunc is a converter function for the config.Map that allows distributions
// (in the future components as well) to build backwards compatible config converters.
type ConfigMapConverterFunc func(*config.Map) error

// NewDefaultConfigProvider returns the default ConfigProvider, and it creates configuration from a file
// defined by the given configFile and overwrites fields using properties.
func NewDefaultConfigProvider(configFileName string, properties []string) ConfigProvider {
	return NewConfigProvider(
		[]configmapprovider.Provider{configmapprovider.NewFile(configFileName), configmapprovider.NewProperties(properties)},
		[]ConfigMapConverterFunc{configprovider.NewExpandConverter()},
		configunmarshaler.NewDefault())
}

func (cm *configProvider) Get(ctx context.Context, factories component.Factories) (*config.Config, error) {
	// First check if already an active watching, close that if any.
	if err := cm.closeIfNeeded(ctx); err != nil {
		return nil, fmt.Errorf("cannot close previous watch: %w", err)
	}

	var err error
	cm.ret, err = mergeRetrieve(ctx, cm.onChange, cm.configMapProviders)
	if err != nil {
		// Nothing to close, no valid retrieved value.
		cm.ret = nil
		return nil, fmt.Errorf("cannot retrieve the configuration: %w", err)
	}

	var cfgMap *config.Map
	cfgMap, err = cm.ret.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot get the configuration: %w", err)
	}

	// Apply all converters.
	for _, cfgMapConv := range cm.cfgMapConverters {
		if err = cfgMapConv(cfgMap); err != nil {
			return nil, fmt.Errorf("cannot convert the config.Map: %w", err)
		}
	}

	var cfg *config.Config
	if cfg, err = cm.configUnmarshaler.Unmarshal(cfgMap, factories); err != nil {
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
	return multierr.Combine(cm.closeIfNeeded(ctx), mergeShutdown(ctx, cm.configMapProviders))
}

func mergeRetrieve(ctx context.Context, onChange func(*configmapprovider.ChangeEvent), providers []configmapprovider.Provider) (configmapprovider.Retrieved, error) {
	var retrs []configmapprovider.Retrieved
	retCfgMap := config.NewMap()
	for _, p := range providers {
		retr, err := p.Retrieve(ctx, onChange)
		if err != nil {
			return nil, err
		}
		cfgMap, err := retr.Get(ctx)
		if err != nil {
			return nil, err
		}
		if err = retCfgMap.Merge(cfgMap); err != nil {
			return nil, err
		}
		retrs = append(retrs, retr)
	}
	return configmapprovider.NewRetrieved(
		func(ctx context.Context) (*config.Map, error) {
			return retCfgMap, nil
		},
		configmapprovider.WithClose(func(ctxF context.Context) error {
			var err error
			for _, ret := range retrs {
				err = multierr.Append(err, ret.Close(ctxF))
			}
			return err
		}))
}

func mergeShutdown(ctx context.Context, providers []configmapprovider.Provider) error {
	var errs error
	for _, p := range providers {
		errs = multierr.Append(errs, p.Shutdown(ctx))
	}

	return errs
}
