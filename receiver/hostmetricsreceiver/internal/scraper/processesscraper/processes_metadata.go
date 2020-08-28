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

package processesscraper

import (
	"go.opentelemetry.io/collector/internal/dataold"
)

// descriptors

var processesRunningDescriptor = func() dataold.MetricDescriptor {
	descriptor := dataold.NewMetricDescriptor()
	descriptor.InitEmpty()
	descriptor.SetName("system.processes.running")
	descriptor.SetDescription("Total number of running processes.")
	descriptor.SetUnit("1")
	descriptor.SetType(dataold.MetricTypeInt64)
	return descriptor
}()

var processesBlockedDescriptor = func() dataold.MetricDescriptor {
	descriptor := dataold.NewMetricDescriptor()
	descriptor.InitEmpty()
	descriptor.SetName("system.processes.blocked")
	descriptor.SetDescription("Total number of blocked processes.")
	descriptor.SetUnit("1")
	descriptor.SetType(dataold.MetricTypeInt64)
	return descriptor
}()
