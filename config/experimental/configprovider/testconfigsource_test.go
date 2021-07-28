// Copyright Splunk, Inc.
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

package configprovider

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/config/experimental/configsource"
)

// testConfigSource a ConfigSource to be used in tests.
type testConfigSource struct {
	ValueMap map[string]valueEntry

	ErrOnNewSession  error
	ErrOnRetrieve    error
	ErrOnRetrieveEnd error
	ErrOnClose       error

	OnRetrieve func(ctx context.Context, selector string, params interface{}) error
}

type valueEntry struct {
	Value            interface{}
	WatchForUpdateFn func() error
}

var _ (configsource.ConfigSource) = (*testConfigSource)(nil)
var _ (configsource.Session) = (*testConfigSource)(nil)

func (t *testConfigSource) NewSession(context.Context) (configsource.Session, error) {
	if t.ErrOnNewSession != nil {
		return nil, t.ErrOnNewSession
	}
	return t, nil
}

func (t *testConfigSource) Retrieve(ctx context.Context, selector string, params interface{}) (configsource.Retrieved, error) {
	if t.OnRetrieve != nil {
		if err := t.OnRetrieve(ctx, selector, params); err != nil {
			return nil, err
		}
	}

	if t.ErrOnRetrieve != nil {
		return nil, t.ErrOnRetrieve
	}

	entry, ok := t.ValueMap[selector]
	if !ok {
		return nil, fmt.Errorf("no value for selector %q", selector)
	}

	watchForUpdateFn := func() error {
		return configsource.ErrWatcherNotSupported
	}

	if entry.WatchForUpdateFn != nil {
		watchForUpdateFn = entry.WatchForUpdateFn
	}

	return &retrieved{
		value:            entry.Value,
		watchForUpdateFn: watchForUpdateFn,
	}, nil
}

func (t *testConfigSource) RetrieveEnd(context.Context) error {
	return t.ErrOnRetrieveEnd
}

func (t *testConfigSource) Close(context.Context) error {
	return t.ErrOnClose
}
