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

package memoryscraper

import (
	"go.opentelemetry.io/collector/consumer/pdata"
)

// labels

const stateLabelName = "state"

// state label values

const (
	bufferedStateLabelValue          = "buffered"
	cachedStateLabelValue            = "cached"
	inactiveStateLabelValue          = "inactive"
	freeStateLabelValue              = "free"
	slabReclaimableStateLabelValue   = "slab_reclaimable"
	slabUnreclaimableStateLabelValue = "slab_unreclaimable"
	usedStateLabelValue              = "used"
)

// descriptors

var memoryUsageDescriptor = func() pdata.Metric {
	metric := pdata.NewMetric()
	metric.InitEmpty()
	metric.SetName("system.memory.usage")
	metric.SetDescription("Bytes of memory in use.")
	metric.SetUnit("bytes")
	metric.SetDataType(pdata.MetricDataTypeIntSum)
	sum := metric.IntSum()
	sum.InitEmpty()
	sum.SetIsMonotonic(false)
	sum.SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
	return metric
}()
