// Copyright  OpenTelemetry Authors
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

package component

import (
	"context"
	"fmt"

	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"go.uber.org/zap"
)

// PipelineFactory ...
type PipelineFactory interface {
	Type() configmodels.Type

	CreateReceiver(context.Context, ReceiverFactoryBase, *zap.Logger, configmodels.Receiver, Consumer) (Receiver, error)
	CreateExporter(context.Context, ExporterFactoryBase, *zap.Logger, configmodels.Exporter) (Exporter, error)
	CreateProcessor(context.Context, ProcessorFactoryBase, *zap.Logger, Consumer, configmodels.Processor) (Processor, error)

	// todo: use a slightly safer type
	CreateFanOutConnector([]Consumer) Consumer
	CreateCloningFanOutConnector([]Consumer) Consumer
}

// MakePipelinesBuilderMap ...
func MakePipelinesBuilderMap(builders ...PipelineFactory) (map[configmodels.Type]PipelineFactory, error) {
	bMap := map[configmodels.Type]PipelineFactory{}
	for _, b := range builders {
		if _, ok := bMap[b.Type()]; ok {
			return bMap, fmt.Errorf("duplicate extension factory %q", b.Type())
		}
		bMap[b.Type()] = b
	}
	return bMap, nil
}
