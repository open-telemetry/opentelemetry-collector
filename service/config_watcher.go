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

	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configmapprovider"
	"go.opentelemetry.io/collector/config/experimental/configsource"
)

type configWatcher struct {
	cfg     *config.Config
	ret     configmapprovider.Retrieved
	watcher chan error
}

func newConfigWatcher(ctx context.Context, set CollectorSettings) (*configWatcher, error) {
	cm := &configWatcher{watcher: make(chan error, 1)}

	ret, err := set.ConfigMapProvider.Retrieve(ctx, cm.onChange)
	if err != nil {
		return nil, fmt.Errorf("cannot retrieve the configuration: %w", err)
	}

	m, err := ret.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot get the configuration: %w", err)
	}

	//sources, err := set.ConfigUnmarshaler.UnmarshalSources(m, set.Factories)
	//if err != nil {
	//	return nil, fmt.Errorf("cannot unmarshal the configuration: %w", err)
	//}
	//
	//cfgMap := config.NewMap()
	//err := retrieveFromSources(ctx, cfgMap, sources, cm.onChange)
	//if err != nil {
	//	return nil, fmt.Errorf("cannot retrieve the configuration: %w", err)
	//}
	//err := mergeConfigMap(cfgMap, m)

	var cfg *config.Config
	if cfg, err = set.ConfigUnmarshaler.Unmarshal(m, set.Factories); err != nil {
		return nil, fmt.Errorf("cannot unmarshal the configuration: %w", err)
	}

	if err = cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	cm.cfg = cfg
	cm.ret = ret

	return cm, nil
}

//func retrieveFromSources(ctx context.Context, cfgMap *config.Map, sources []component.ConfigSource, onChange func(*configmapprovider.ChangeEvent)) error {
//	for _, source := range sources {
//		retrieved, err := source.Retrieve(ctx, onChange)
//		m, err := retrieved.Get(ctx)
//		err := mergeConfigMap(cfgMap, m)
//	}
//	return nil
//}

func (cm *configWatcher) onChange(event *configmapprovider.ChangeEvent) {
	if event.Error != configsource.ErrSessionClosed {
		cm.watcher <- event.Error
	}
}

func (cm *configWatcher) close(ctx context.Context) error {
	close(cm.watcher)
	return cm.ret.Close(ctx)
}
