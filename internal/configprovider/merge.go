// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package configprovider // import "go.opentelemetry.io/collector/internal/configprovider"

import (
	"context"

	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configmapprovider"
)

type mergeMapProvider struct {
	providers []configmapprovider.Provider
}

// NewMerge returns a Provider, that merges the result from multiple Provider.
//
// The ConfigMaps are merged in the given order, by merging all of them in order into an initial empty map.
func NewMerge(ps ...configmapprovider.Provider) configmapprovider.Provider {
	return &mergeMapProvider{providers: ps}
}

func (mp *mergeMapProvider) Retrieve(ctx context.Context, onChange func(*configmapprovider.ChangeEvent)) (configmapprovider.Retrieved, error) {
	var retrs []configmapprovider.Retrieved
	retCfgMap := config.NewMap()
	for _, p := range mp.providers {
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

func (mp *mergeMapProvider) Shutdown(ctx context.Context) error {
	var errs error
	for _, p := range mp.providers {
		errs = multierr.Append(errs, p.Shutdown(ctx))
	}

	return errs
}
