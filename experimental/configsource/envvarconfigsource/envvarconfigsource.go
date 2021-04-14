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

package envvarconfigsource

import (
	"context"
	"os"

	"go.opentelemetry.io/collector/experimental/configsource"
)

type envVarConfigSource struct{}

var _ (configsource.ConfigSource) = (*envVarConfigSource)(nil)

func (e *envVarConfigSource) NewSession(context.Context) (configsource.Session, error) {
	return &envVarSession{}, nil
}

type envVarSession struct{}

func (e *envVarSession) Retrieve(_ context.Context, selector string, _ interface{}) (configsource.Retrieved, error) {
	v, _ := os.LookupEnv(selector)
	return configsource.NewRetrieved(v, configsource.WatcherNotSupported), nil
}

func (e *envVarSession) RetrieveEnd(context.Context) error {
	return nil
}

func (e *envVarSession) Close(context.Context) error {
	return nil
}
