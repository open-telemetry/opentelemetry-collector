// Copyright 2020 The OpenTelemetry Authors
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

package kafkaexporter

import (
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"

	"go.opentelemetry.io/collector/obsreport"
)

var (
	tagInstanceName, _ = tag.NewKey("name")

	statSendSuccess = stats.Int64("producer_success", "Number of times the Kafka producer failed to send a message", stats.UnitDimensionless)
	statSendErr     = stats.Int64("producer_fail", "Number of times the Kafka producer successfully send a message", stats.UnitDimensionless)
)

func MetricViews() []*view.View {
	tagKeys := []tag.Key{tagInstanceName}

	countSendSuccess := &view.View{
		Name:        statSendSuccess.Name(),
		Measure:     statSendSuccess,
		Description: statSendSuccess.Description(),
		TagKeys:     tagKeys,
		Aggregation: view.Sum(),
	}

	countSendErr := &view.View{
		Name:        statSendErr.Name(),
		Measure:     statSendErr,
		Description: statSendErr.Description(),
		TagKeys:     tagKeys,
		Aggregation: view.Sum(),
	}

	views := []*view.View{
		countSendSuccess,
		countSendErr,
	}
	return obsreport.ProcessorMetricViews(typeStr, views)
}
