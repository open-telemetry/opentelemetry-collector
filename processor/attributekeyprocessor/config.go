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

package attributekeyprocessor

import (
	"github.com/open-telemetry/opentelemetry-service/config/configmodels"
)

// Config defines configuration for Attribute Key processor.
type Config struct {
	configmodels.ProcessorSettings `mapstructure:",squash"`
	// map key is the attribute key to be replaced.
	Keys map[string]NewKeyProperties `mapstructure:"keys"`
}

// NewKeyProperties defines the key's replacments properties.
type NewKeyProperties struct {
	// NewKey is the value that will be used as the new key for the attribute.
	NewKey string `mapstructure:"replacement"`
	// Overwrite is set to true to indicate that the replacement should be
	// performed even if the new key already exists on the attributes.
	// In this case the original value associated with the new key is lost.
	Overwrite bool `mapstructure:"overwrite"`
	// KeepOriginal is set to true to indicate that the original key
	// should not be removed from the attributes.
	KeepOriginal bool `mapstructure:"keep"`
}
