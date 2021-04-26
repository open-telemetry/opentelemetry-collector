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

package internal

import (
	"go.opencensus.io/metric"
	"go.opencensus.io/metric/metricdata"
	"go.opencensus.io/metric/metricproducer"
)

var (
	metricsRegistry = metric.NewRegistry()

	upGauge, err = metricsRegistry.AddFloat64Gauge(
		"exporter/up",
		metric.WithDescription("Whether the endpoint is alive or not"),
		metric.WithLabelKeys("instance"),
		metric.WithUnit(metricdata.UnitDimensionless))
)

func init() {
	if err != nil {
		panic(err)
	}
	metricproducer.GlobalManager().AddProducer(metricsRegistry)
}

func recordInstanceAsUp(instanceValue string) {
	ent, err := upGauge.GetEntry(metricdata.NewLabelValue(instanceValue))
	if err != nil {
		panic(err)
	}
	ent.Set(1)
}

func recordInstanceAsDown(instanceValue string) {
	ent, err := upGauge.GetEntry(metricdata.NewLabelValue(instanceValue))
	if err != nil {
		panic(err)
	}
	ent.Set(0)
}
