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

package extension

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-service/config/configmodels"
)

// Factory is a factory interface for extensions to the service.
type Factory interface {
	// Type gets the type of the extension created by this factory.
	Type() string

	// CreateDefaultConfig creates the default configuration for the extension.
	CreateDefaultConfig() configmodels.Extension

	// CreateExtension creates a service extension based on the given config.
	CreateExtension(logger *zap.Logger, cfg configmodels.Extension) (ServiceExtension, error)
}

// Build takes a list of extension factories and returns a map of type map[string]Factory
// with factory type as keys. It returns a non-nil error when more than one factories
// have the same type.
func Build(factories ...Factory) (map[string]Factory, error) {
	fMap := map[string]Factory{}
	for _, f := range factories {
		if _, ok := fMap[f.Type()]; ok {
			return fMap, fmt.Errorf("duplicate extension factory %q", f.Type())
		}
		fMap[f.Type()] = f
	}
	return fMap, nil
}
