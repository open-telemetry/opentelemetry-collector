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

package kafkaexporter

import (
	"context"
	"fmt"

	"github.com/spf13/viper"
	kafka "github.com/yancl/opencensus-go-exporter-kafka"

	"github.com/census-instrumentation/opencensus-service/data"
	"github.com/census-instrumentation/opencensus-service/exporter"
	"github.com/census-instrumentation/opencensus-service/exporter/exporterparser"
)

type kafkaConfig struct {
	Brokers []string `mapstructure:"brokers,omitempty"`
	Topic   string   `mapstructure:"topic,omitempty"`
}

type kafkaExporter struct {
	exporter *kafka.Exporter
}

var _ exporter.TraceExporter = (*kafkaExporter)(nil)

// KafkaExportersFromViper unmarshals the viper and returns an exporter.TraceExporter targeting
// Kafka according to the configuration settings.
func KafkaExportersFromViper(v *viper.Viper) (tes []exporter.TraceExporter, mes []exporter.MetricsExporter, doneFns []func() error, err error) {
	var cfg struct {
		Kafka *kafkaConfig `mapstructure:"kafka"`
	}

	if err := v.Unmarshal(&cfg); err != nil {
		return nil, nil, nil, err
	}
	kc := cfg.Kafka
	if kc == nil {
		return nil, nil, nil, nil
	}

	kde, kerr := kafka.NewExporter(kafka.Options{
		Brokers: kc.Brokers,
		Topic:   kc.Topic,
	})

	if kerr != nil {
		return nil, nil, nil, fmt.Errorf("Cannot configure Kafka Trace exporter: %v", kerr)
	}

	tes = append(tes, &kafkaExporter{exporter: kde})
	doneFns = append(doneFns, func() error {
		kde.Flush()
		return nil
	})
	return tes, nil, doneFns, nil
}

func (kde *kafkaExporter) ExportSpans(ctx context.Context, td data.TraceData) error {
	return exporterparser.OcProtoSpansToOCSpanDataInstrumented(ctx, "kafka", kde.exporter, td)
}
