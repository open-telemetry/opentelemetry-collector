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

package configmapprovider // import "go.opentelemetry.io/collector/config/configmapprovider"

import (
	"context"

	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/config"
)

// TODO: Add support to "merge" watchable interface.

type mergeMapProvider struct {
	providers []MapProvider
}

// NewMergeMapProvider returns a MapProvider, that merges the result from multiple MapProvider.
//
// The ConfigMaps are merged in the given order, by merging all of them in order into an initial empty map.
func NewMergeMapProvider(ps ...MapProvider) MapProvider {
	return &mergeMapProvider{providers: ps}
}

func (mp *mergeMapProvider) Retrieve(ctx context.Context) (Retrieved, error) {
	retCfgMap := config.NewMap()
	for _, p := range mp.providers {
		retr, err := p.Retrieve(ctx)
		if err != nil {
			return nil, err
		}
		if err = retCfgMap.Merge(retr.Get()); err != nil {
			return nil, err
		}
	}
	return &simpleRetrieved{confMap: retCfgMap}, nil
}

func (mp *mergeMapProvider) Close(ctx context.Context) error {
	var errs error
	for _, p := range mp.providers {
		errs = multierr.Append(errs, p.Close(ctx))
	}

	return errs
}
