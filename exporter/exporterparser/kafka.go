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
	"context"
	"fmt"

	"github.com/yancl/opencensus-go-exporter-kafka"
	"go.opencensus.io/trace"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	"github.com/census-instrumentation/opencensus-service/exporter"
)

type kafkaConfig struct {
	Brokers []string `yaml:"brokers,omitempty"`
	Topic   string   `yaml:"topic,omitempty"`
}

type kafkaExporter struct {
	exporter *kafka.Exporter
}

var _ exporter.TraceExporter = (*kafkaExporter)(nil)

// KafkaExportersFromYAML parses the yaml bytes and returns an exporter.TraceExporter targeting
// Kafka according to the configuration settings.
func KafkaExportersFromYAML(config []byte) (tes []exporter.TraceExporter, doneFns []func() error, err error) {
	var cfg struct {
		Exporters *struct {
			Kafka *kafkaConfig `yaml:"kafka"`
		} `yaml:"exporters"`
	}

	if err := yamlUnmarshal(config, &cfg); err != nil {
		return nil, nil, err
	}
	if cfg.Exporters == nil {
		return nil, nil, nil
	}
	kc := cfg.Exporters.Kafka
	if kc == nil {
		return nil, nil, nil
	}

	kde, kerr := kafka.NewExporter(kafka.Options{
		Brokers: kc.Brokers,
		Topic:   kc.Topic,
	})

	if kerr != nil {
		return nil, nil, fmt.Errorf("Cannot configure Kafka Trace exporter: %v", kerr)
	}

	tes = append(tes, &kafkaExporter{exporter: kde})
	doneFns = append(doneFns, func() error {
		kde.Flush()
		return nil
	})
	return tes, doneFns, nil
}

func (kde *kafkaExporter) ExportSpanData(ctx context.Context, node *commonpb.Node, spandata ...*trace.SpanData) error {
	return exportSpans(ctx, node, "kafka", kde.exporter, spandata)
}
