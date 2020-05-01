// Copyright 2020, OpenTelemetry Authors
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

package memoryscraper

import (
	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
)

// memory metric constants

const stateLabelName = "state"

const (
	usedStateLabelValue              = "used"
	bufferedStateLabelValue          = "buffered"
	cachedStateLabelValue            = "cached"
	freeStateLabelValue              = "free"
	slabReclaimableStateLabelValue   = "slab_reclaimable"
	slabUnreclaimableStateLabelValue = "slab_unreclaimable"
)

var metricMemoryUsedDescriptor = createMetricMemoryUsedDescriptor()

func createMetricMemoryUsedDescriptor() pdata.MetricDescriptor {
	descriptor := pdata.NewMetricDescriptor()
	descriptor.InitEmpty()
	descriptor.SetName("host/memory/used")
	descriptor.SetDescription("Bytes of memory in use.")
	descriptor.SetUnit("bytes")
	descriptor.SetType(pdata.MetricTypeGaugeInt64)
	return descriptor
}
