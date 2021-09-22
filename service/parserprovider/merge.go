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

package parserprovider

import (
	"context"

	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer/consumererror"
)

// TODO: Add support to "merge" watchable interface.

type mergeProvider struct {
	providers []ParserProvider
}

// NewMergeProvider returns a config.ParserProvider, that merges the result from multiple ParserProvider.
//
// The ConfigMaps are merged in the given order, by merging all of them in order into an initial empty map.
func NewMergeProvider(ps ...ParserProvider) ParserProvider {
	return &mergeProvider{providers: ps}
}

func (mp *mergeProvider) Get(ctx context.Context) (*config.Map, error) {
	ret := config.NewMap()
	for _, p := range mp.providers {
		cfgMap, err := p.Get(ctx)
		if err != nil {
			return nil, err
		}
		if err = ret.Merge(cfgMap); err != nil {
			return nil, err
		}
	}
	return ret, nil
}

func (mp *mergeProvider) Close(ctx context.Context) error {
	var errs []error
	for _, p := range mp.providers {
		if err := p.Close(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	return consumererror.Combine(errs)
}
