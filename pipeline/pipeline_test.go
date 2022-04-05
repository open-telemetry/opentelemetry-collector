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

package pipeline // import "go.opentelemetry.io/collector/pipeline"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/exporter/loggingexporter"
	"go.opentelemetry.io/collector/exporter/otlpexporter"
	"go.opentelemetry.io/collector/processor/batchprocessor"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
	"go.opentelemetry.io/otel/metric/nonrecording"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

func components() (component.Factories, error) {
	var err error
	factories := component.Factories{}

	factories.Receivers, err = component.MakeReceiverFactoryMap(
		otlpreceiver.NewFactory(),
	)
	if err != nil {
		return component.Factories{}, err
	}

	factories.Exporters, err = component.MakeExporterFactoryMap(
		loggingexporter.NewFactory(),
		otlpexporter.NewFactory(),
	)
	if err != nil {
		return component.Factories{}, err
	}

	factories.Processors, err = component.MakeProcessorFactoryMap(
		batchprocessor.NewFactory(),
	)
	if err != nil {
		return component.Factories{}, err
	}

	return factories, nil
}

func ExampleNewBuilder() {

	receiverFactory := otlpreceiver.NewFactory()
	receiverCfg := receiverFactory.CreateDefaultConfig().(*otlpreceiver.Config)
	receiverCfg.HTTP = nil // I need to explicitly nil HTTP, since this is done in Unmarshal usually
	receiverCfg.GRPC = &configgrpc.GRPCServerSettings{
		NetAddr: confignet.NetAddr{
			Transport: "tcp",
			// I can't really express 'use the default gRPC settings here' as one can do by setting 'grpc:' on the YAML
			Endpoint: "0.0.0.0:4317",
		},
		// I only know this by reading the code
		ReadBufferSize: 512 * 1024,
	}

	processorFactory := batchprocessor.NewFactory()
	processorCfg := processorFactory.CreateDefaultConfig()

	exporterFactory := loggingexporter.NewFactory()
	exporterCfg := exporterFactory.CreateDefaultConfig().(*loggingexporter.Config)

	components, err := components()
	if err != nil {
		panic(err)
	}

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	builder := NewBuilder(
		component.TelemetrySettings{
			Logger:         logger,
			MeterProvider:  nonrecording.NewNoopMeterProvider(),
			TracerProvider: trace.NewNoopTracerProvider(),
		},
		component.NewDefaultBuildInfo(),
		components,
	)

	pipeline, err := builder.BuildMetricsPipeline(
		context.TODO(),
		receiverCfg,
		[]config.Processor{processorCfg},
		exporterCfg,
	)
	if err != nil {
		panic(err)
	}

	err = pipeline.Run(context.TODO())
	if err != nil {
		panic(err)
	}
}
