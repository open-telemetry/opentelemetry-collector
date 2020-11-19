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

package testbed

// MockDataConsumer is an interface that keeps the count of number of events received by mock receiver.
// This is mainly useful for the Exporters that are not have the matching receiver
type MockTraceDataConsumer interface {
	// MockConsumeTraceData receives traces and counts the number of events received.
	MockConsumeTraceData(spansCount int) error
}

type MockMetricDataConsumer interface {
	// MockConsumeMetricData receives metrics and counts the number of events received.
	MockConsumeMetricData(metricsCount int) error
}
