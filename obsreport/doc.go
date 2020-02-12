// Copyright 2020 OpenTelemetry Authors
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

// Package obsreport provides unified and consistent observability signals (
// metrics, tracing, etc) for components of the OpenTelemetry collector.
//
// The function Configure is used to control which signals are going to be
// generated. It provides functions for the typical operations of receivers,
// processors, and exporters.
//
// Receivers should use the respective start and end according to the data type
// being received, ie.:
//
// 	* TraceData receive operations should use the pair:
// 		StartTraceDataReceiveOp/EndTraceDataReceiveOp
//
// 	* Metrics receive operations should use the pair:
// 		StartMetricsReceiveOp/EndMetricsReceiveOp
//
// Similar for exporters:
//
// 	* TraceData export operations should use the pair:
// 		StartTraceDataExportOp/EndTraceDataExportOp
//
// 	* Metrics export operations should use the pair:
// 		StartMetricsExportOp/EndMetricsExportOp
//
// The package is capable of generating legacy metrics by using the
// observability package allowing a controlled transition from legacy to the
// new metrics. The main differences regarding the legacy metrics are:
//
// 1. "Amount of metric data" is measured as metric points (ie.: a single value
// in time), contrast it with number of time series used legacy. Number of
// metric data points is a more general concept regarding various metric
// formats.
//
// 2. Exporters measure the number of items, ie.: number of spans or metric
// points, that were sent and the ones for which the attempt to send failed.
// Legacy treated the latter as "dropped" data but depending on configuration
// it is not certain that the data was actually dropped, ie.: there may be a
// processor, like the "queued_retry", taking care of a retry.
//
// 3. All measurements of "amount of data" used in the new metrics should
// reflect their native formats, not internal formats of the Collector. This is
// to facilitate reconciliation between the Collector, client and backend.
package obsreport
