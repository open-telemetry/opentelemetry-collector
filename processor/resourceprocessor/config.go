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

package resourceprocessor

import (
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

// Config defines configuration for Resource processor.
type Config struct {
	configmodels.ProcessorSettings `mapstructure:",squash"`

	// AttributesActions specifies the list of actions to be applied on resource attributes.
	// The set of actions are {INSERT, UPDATE, UPSERT, DELETE, HASH, EXTRACT}.
	AttributesActions []processorhelper.ActionKeyValue `mapstructure:"attributes"`

	// ResourceType field is deprecated. Set "opencensus.type" key in "attributes.upsert" map instead.
	ResourceType string `mapstructure:"type"`

	// Deprecated: Use "attributes.upsert" instead.
	Labels map[string]string `mapstructure:"labels"`
}
