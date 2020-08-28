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

package swapscraper

import (
	"go.opentelemetry.io/collector/internal/dataold"
)

// labels

const (
	deviceLabelName    = "device"
	directionLabelName = "direction"
	stateLabelName     = "state"
	typeLabelName      = "type"
)

// direction label values

const (
	inDirectionLabelValue  = "page_in"
	outDirectionLabelValue = "page_out"
)

// state label values

const (
	cachedLabelValue = "cached"
	freeLabelValue   = "free"
	usedLabelValue   = "used"
)

// type label values

const (
	majorTypeLabelValue = "major"
	minorTypeLabelValue = "minor"
)

var swapUsageDescriptor = func() dataold.MetricDescriptor {
	descriptor := dataold.NewMetricDescriptor()
	descriptor.InitEmpty()
	descriptor.SetName("system.swap.usage")
	descriptor.SetDescription("Swap (unix) or pagefile (windows) usage.")
	descriptor.SetUnit("pages")
	descriptor.SetType(dataold.MetricTypeInt64)
	return descriptor
}()

var swapPagingDescriptor = func() dataold.MetricDescriptor {
	descriptor := dataold.NewMetricDescriptor()
	descriptor.InitEmpty()
	descriptor.SetName("system.swap.paging_ops")
	descriptor.SetDescription("The number of paging operations.")
	descriptor.SetUnit("1")
	descriptor.SetType(dataold.MetricTypeMonotonicInt64)
	return descriptor
}()

var swapPageFaultsDescriptor = func() dataold.MetricDescriptor {
	descriptor := dataold.NewMetricDescriptor()
	descriptor.InitEmpty()
	descriptor.SetName("system.swap.page_faults")
	descriptor.SetDescription("The number of page faults.")
	descriptor.SetUnit("1")
	descriptor.SetType(dataold.MetricTypeMonotonicInt64)
	return descriptor
}()
