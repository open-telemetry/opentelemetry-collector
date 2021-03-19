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

package builder

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/extension/extensionhelper"
	"go.opentelemetry.io/collector/internal/testcomponents"
	"go.opentelemetry.io/collector/processor/processorhelper"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
)

func createTestFactories() component.Factories {
	exampleReceiverFactory := testcomponents.ExampleReceiverFactory
	exampleProcessorFactory := testcomponents.ExampleProcessorFactory
	exampleExporterFactory := testcomponents.ExampleExporterFactory
	badExtensionFactory := newBadExtensionFactory()
	badReceiverFactory := newBadReceiverFactory()
	badProcessorFactory := newBadProcessorFactory()
	badExporterFactory := newBadExporterFactory()

	factories := component.Factories{
		Extensions: map[configmodels.Type]component.ExtensionFactory{
			badExtensionFactory.Type(): badExtensionFactory,
		},
		Receivers: map[configmodels.Type]component.ReceiverFactory{
			exampleReceiverFactory.Type(): exampleReceiverFactory,
			badReceiverFactory.Type():     badReceiverFactory,
		},
		Processors: map[configmodels.Type]component.ProcessorFactory{
			exampleProcessorFactory.Type(): exampleProcessorFactory,
			badProcessorFactory.Type():     badProcessorFactory,
		},
		Exporters: map[configmodels.Type]component.ExporterFactory{
			exampleExporterFactory.Type(): exampleExporterFactory,
			badExporterFactory.Type():     badExporterFactory,
		},
	}

	return factories
}

func newBadReceiverFactory() component.ReceiverFactory {
	return receiverhelper.NewFactory("bf", func() configmodels.Receiver {
		return &configmodels.ReceiverSettings{
			TypeVal: "bf",
		}
	})
}

func newBadProcessorFactory() component.ProcessorFactory {
	return processorhelper.NewFactory("bf", func() configmodels.Processor {
		return &configmodels.ProcessorSettings{
			TypeVal: "bf",
		}
	})
}

func newBadExporterFactory() component.ExporterFactory {
	return exporterhelper.NewFactory("bf", func() configmodels.Exporter {
		return &configmodels.ExporterSettings{
			TypeVal: "bf",
		}
	})
}

func newBadExtensionFactory() component.ExtensionFactory {
	return extensionhelper.NewFactory(
		"bf",
		func() configmodels.Extension {
			return &configmodels.ExporterSettings{
				TypeVal: "bf",
			}
		},
		func(ctx context.Context, params component.ExtensionCreateParams, extension configmodels.Extension) (component.Extension, error) {
			return nil, nil
		},
	)
}
