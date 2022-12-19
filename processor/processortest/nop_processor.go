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

package processortest // import "go.opentelemetry.io/collector/processor/processortest"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/processor"
)

const typeStr = "nop"

// NewNopCreateSettings returns a new nop settings for Create* functions.
var NewNopCreateSettings = componenttest.NewNopProcessorCreateSettings //nolint:staticcheck

// NewNopFactory returns a component.ProcessorFactory that constructs nop processors.
var NewNopFactory = componenttest.NewNopProcessorFactory //nolint:staticcheck

// NewNopBuilder returns a processor.Builder that constructs nop receivers.
func NewNopBuilder() *processor.Builder {
	nopFactory := NewNopFactory()
	return processor.NewBuilder(
		map[component.ID]component.Config{component.NewID(typeStr): nopFactory.CreateDefaultConfig()},
		map[component.Type]processor.Factory{typeStr: nopFactory})
}
