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

// package observ contains logic pertaining to the internal observation
// of the fluent forward receiver.
package observ

import (
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
)

var (
	ConnectionsOpened = stats.Int64(
		"fluent_opened_connections",
		"Number of connections opened to the fluentforward receiver",
		stats.UnitDimensionless)
	connectionsOpenedView = &view.View{
		Name:        ConnectionsOpened.Name(),
		Measure:     ConnectionsOpened,
		Description: ConnectionsOpened.Description(),
		Aggregation: view.Sum(),
	}

	ConnectionsClosed = stats.Int64(
		"fluent_closed_connections",
		"Number of connections closed to the fluentforward receiver",
		stats.UnitDimensionless)
	connectionsClosedView = &view.View{
		Name:        ConnectionsClosed.Name(),
		Measure:     ConnectionsClosed,
		Description: ConnectionsClosed.Description(),
		Aggregation: view.Sum(),
	}

	EventsParsed = stats.Int64(
		"fluent_events_parsed",
		"Number of Fluent events parsed successfully",
		stats.UnitDimensionless)
	eventsParsedView = &view.View{
		Name:        EventsParsed.Name(),
		Measure:     EventsParsed,
		Description: EventsParsed.Description(),
		Aggregation: view.Sum(),
	}

	FailedToParse = stats.Int64(
		"fluent_parse_failures",
		"Number of times Fluent messages failed to be decoded",
		stats.UnitDimensionless)
	failedToParseView = &view.View{
		Name:        FailedToParse.Name(),
		Measure:     FailedToParse,
		Description: FailedToParse.Description(),
		Aggregation: view.Sum(),
	}

	RecordsGenerated = stats.Int64(
		"fluent_records_generated",
		"Number of log records generated from Fluent forward input",
		stats.UnitDimensionless)
	recordsGeneratedView = &view.View{
		Name:        RecordsGenerated.Name(),
		Measure:     RecordsGenerated,
		Description: RecordsGenerated.Description(),
		Aggregation: view.Sum(),
	}
)

func MetricViews() []*view.View {
	return []*view.View{
		connectionsOpenedView,
		connectionsClosedView,
		eventsParsedView,
		failedToParseView,
		recordsGeneratedView,
	}
}
