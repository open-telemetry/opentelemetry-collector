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
	"go.opentelemetry.io/collector/config"
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
		Extensions: map[config.Type]component.ExtensionFactory{
			badExtensionFactory.Type(): badExtensionFactory,
		},
		Receivers: map[config.Type]component.ReceiverFactory{
			exampleReceiverFactory.Type(): exampleReceiverFactory,
			badReceiverFactory.Type():     badReceiverFactory,
		},
		Processors: map[config.Type]component.ProcessorFactory{
			exampleProcessorFactory.Type(): exampleProcessorFactory,
			badProcessorFactory.Type():     badProcessorFactory,
		},
		Exporters: map[config.Type]component.ExporterFactory{
			exampleExporterFactory.Type(): exampleExporterFactory,
			badExporterFactory.Type():     badExporterFactory,
		},
	}

	return factories
}

func newBadReceiverFactory() component.ReceiverFactory {
	return receiverhelper.NewFactory("bf", func() config.Receiver {
		return &struct {
			config.ReceiverSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
		}{
			ReceiverSettings: config.NewReceiverSettings(config.NewID("bf")),
		}
	})
}

func newBadProcessorFactory() component.ProcessorFactory {
	return processorhelper.NewFactory("bf", func() config.Processor {
		return &struct {
			config.ProcessorSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
		}{
			ProcessorSettings: config.NewProcessorSettings(config.NewID("bf")),
		}
	})
}

func newBadExporterFactory() component.ExporterFactory {
	return exporterhelper.NewFactory("bf", func() config.Exporter {
		return &struct {
			config.ExporterSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
		}{
			ExporterSettings: config.NewExporterSettings(config.NewID("bf")),
		}
	})
}

func newBadExtensionFactory() component.ExtensionFactory {
	return extensionhelper.NewFactory(
		"bf",
		func() config.Extension {
			return &struct {
				config.ExtensionSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
			}{
				ExtensionSettings: config.NewExtensionSettings(config.NewID("bf")),
			}
		},
		func(ctx context.Context, set component.ExtensionCreateSettings, extension config.Extension) (component.Extension, error) {
			return nil, nil
		},
	)
}
