// Copyright 2019, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package zpages

import (
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-service/config/configmodels"
)

// Factory is the factory for Z pages server.
type Factory struct {
}

// CreateDefaultConfig creates the default configuration for Z Pages.
func (f *Factory) CreateDefaultConfig() configmodels.ZPages {
	return &configmodels.ZPagesSettings{Port: 55679}
}

// CreateZPagesServer creates a Z pages Server based on this config.
func (f *Factory) CreateZPagesServer(
	logger *zap.Logger,
	asyncErrorChannel chan<- error,
	cfg configmodels.ZPages,
) (closeFn func() error, err error) {
	zCfg := cfg.(*configmodels.ZPagesSettings)
	if !zCfg.IsEnabled() {
		return nil, nil
	}
	return Run(logger, asyncErrorChannel, zCfg)
}
