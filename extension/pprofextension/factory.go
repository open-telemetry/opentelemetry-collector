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

package pprofextension

import (
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-service/config/configmodels"
	"github.com/open-telemetry/opentelemetry-service/extension"
)

const (
	// The value of extension "type" in configuration.
	typeStr = "pprof"
)

// Factory is the factory for the extension.
type Factory struct {
}

var _ (extension.Factory) = (*Factory)(nil)

// Type gets the type of the config created by this factory.
func (f *Factory) Type() string {
	return typeStr
}

// CreateDefaultConfig creates the default configuration for the extension.
func (f *Factory) CreateDefaultConfig() configmodels.Extension {
	return &Config{}
}

// CreateExtension creates the extension based on this config.
func (f *Factory) CreateExtension(
	logger *zap.Logger,
	cfg configmodels.Extension,
) (extension.ServiceExtension, error) {
	return nil, nil
}


