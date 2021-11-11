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
	"errors"
	"fmt"
	"sync"

	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configmapprovider"
	"go.opentelemetry.io/collector/config/experimental/configsource"
)

type configWatcher struct {
	cfg     *config.Config
	ret     configmapprovider.Retrieved
	watcher chan error
	stopWG  sync.WaitGroup
}

func newConfigWatcher(ctx context.Context, set CollectorSettings) (*configWatcher, error) {
	ret, err := set.ConfigMapProvider.Retrieve(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot retrieve the configuration: %w", err)
	}

	var cfg *config.Config
	if cfg, err = set.ConfigUnmarshaler.Unmarshal(ret.Get(), set.Factories); err != nil {
		return nil, fmt.Errorf("cannot unmarshal the configuration: %w", err)
	}

	if err = cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	cm := &configWatcher{cfg: cfg, ret: ret, watcher: make(chan error, 1)}
	// If the retrieved value is watchable start a goroutine watching for updates.
	if watchable, ok := ret.(configmapprovider.WatchableRetrieved); ok {
		cm.stopWG.Add(1)
		go func() {
			defer cm.stopWG.Done()
			err = watchable.WatchForUpdate()
			if errors.Is(err, configsource.ErrSessionClosed) {
				// This is the case of shutdown of the whole collector server, nothing to do.
				return
			}
			cm.watcher <- err
		}()
	}

	return cm, nil
}

func (cm *configWatcher) close(ctx context.Context) error {
	defer func() { close(cm.watcher) }()
	// If the retrieved value is watchable start a goroutine watching for updates.
	if watchable, ok := cm.ret.(configmapprovider.WatchableRetrieved); ok {
		return watchable.Close(ctx)
	}

	cm.stopWG.Wait()
	return nil
}
