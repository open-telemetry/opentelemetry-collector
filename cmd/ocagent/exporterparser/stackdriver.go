// Copyright 2018, OpenCensus Authors
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

package exporterparser

import (
	"log"

	"contrib.go.opencensus.io/exporter/stackdriver"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
	yaml "gopkg.in/yaml.v2"
)

type stackdriverConfig struct {
	Stackdriver *struct {
		ProjectID     string `yaml:"project,omitempty"`
		EnableTracing bool   `yaml:"enable_tracing,omitempty"`
	} `yaml:"stackdriver,omitempty"`
}

type stackdriverExporter struct{}

func (s *stackdriverExporter) MakeExporters(config []byte) (se view.Exporter, te trace.Exporter, closer func()) {
	var c stackdriverConfig
	if err := yaml.Unmarshal(config, &c); err != nil {
		log.Fatalf("Cannot unmarshal data: %v", err)
	}
	sc := c.Stackdriver
	if sc == nil {
		return nil, nil, nil
	}
	if !sc.EnableTracing {
		return nil, nil, nil
	}

	// TODO(jbd): Add monitored resources.
	if sc.ProjectID == "" {
		log.Fatal("Stackdriver config requires a project ID")
	}
	sde, err := stackdriver.NewExporter(stackdriver.Options{
		ProjectID: sc.ProjectID,
	})
	if err != nil {
		log.Fatalf("Cannot configure Stackdriver exporter: %v", err)
	}

	closer = sde.Flush
	return nil, sde, closer
}
