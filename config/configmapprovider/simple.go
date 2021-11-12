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

package configmapprovider // import "go.opentelemetry.io/collector/config/configmapprovider"

import (
	"context"

	"go.opentelemetry.io/collector/config"
)

// TODO: This probably will make sense to be exported, but needs better name and documentation.
type simpleRetrieved struct {
	confMap *config.Map
}

func (sr *simpleRetrieved) Get(ctx context.Context) (*config.Map, error) {
	return sr.confMap, nil
}

func (sr *simpleRetrieved) Close(ctx context.Context) error {
	return nil
}
