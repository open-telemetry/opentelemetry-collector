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

package hwexporter

import (
	"context"
	"fmt"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"

	"github.com/open-telemetry/opentelemetry-collector/pipelines/hwpipeline"
)

type Config struct {
	configmodels.ExporterSettings `mapstructure:",squash"`
}

const (
	typeStr = "hw"
)

type Factory struct {
}

var _ hwpipeline.ExporterFactory = (*Factory)(nil)

// Type gets the type of the config created by this factory.
func (f *Factory) Type() configmodels.Type {
	return typeStr
}

// CreateDefaultConfig creates the default configuration for processor.
func (f *Factory) CreateDefaultConfig() configmodels.Exporter {
	return &Config{
		ExporterSettings: configmodels.ExporterSettings{
			TypeVal: typeStr,
			NameVal: typeStr,
		},
	}
}

// CreateTraceProcessor creates a trace processor based on this config.
func (f *Factory) CreateHWExporter() hwpipeline.Exporter {
	return &hwExporter{}
}

type hwExporter struct{}

func (p *hwExporter) ConsumeHW(hw hwpipeline.HelloWorld) error {
	fmt.Println("exporter received::  ", hw)
	return nil
}

func (p *hwExporter) Start(_ context.Context, _ component.Host) error {
	return nil
}

func (p *hwExporter) Shutdown(context.Context) error {
	return nil
}
