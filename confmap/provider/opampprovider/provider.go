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

package opampprovider

import (
	"context"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/internal"
)

const schemeName = "opamp"

type provider struct{}

func New() confmap.Provider {
	return &provider{}
}

func (fmp *provider) Retrieve(_ context.Context, uri string, _ confmap.WatcherFunc) (*confmap.Retrieved, error) {
	// Retrieve config from OpAMP server synchronously.
	// Can also check if a copy exists in the local cache return it and retrieve from OpAMP
	// server asynchronously, then trigger WatcherFunc.
	content := []byte{}
	return internal.NewRetrievedFromYAML(content)
}

func (*provider) Scheme() string {
	return schemeName
}

func (*provider) Shutdown(context.Context) error {
	return nil
}

func (*provider) ConfigUpdateFailed(lastKnownGoodEffectiveConfig, failedEffectiveConfig []byte, failureReason error) {
	// OpAMP Provider can report RemoteConfigStatus.FAILED here
	// if previously an update was triggered via remote config.
}

func (*provider) ConfigUpdateSucceeded(newEffectiveConfig []byte) {
	// OpAMP Provider can report EffectiveConfig to OpAMP server and also report
	// RemoteConfigStatus.APPLIED if previously an update was triggered via remote config.
	// It can also cache locally the last value it got from the OpAMP Server and use during the next startup.
}
