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

package virtualmemoryscraper

import (
	"go.opentelemetry.io/collector/consumer/pdata"
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

var metricSwapUsageDescriptor = createMetricSwapUsageDescriptor()

func createMetricSwapUsageDescriptor() pdata.MetricDescriptor {
	descriptor := pdata.NewMetricDescriptor()
	descriptor.InitEmpty()
	descriptor.SetName("host/swap/usage")
	descriptor.SetDescription("Swap (unix) or pagefile (windows) usage.")
	descriptor.SetUnit("pages")
	descriptor.SetType(pdata.MetricTypeInt64)
	return descriptor
}

var metricPagingDescriptor = createMetricPagingDescriptor()

func createMetricPagingDescriptor() pdata.MetricDescriptor {
	descriptor := pdata.NewMetricDescriptor()
	descriptor.InitEmpty()
	descriptor.SetName("host/swap/paging")
	descriptor.SetDescription("The number of paging operations.")
	descriptor.SetUnit("1")
	descriptor.SetType(pdata.MetricTypeMonotonicInt64)
	return descriptor
}

var metricPageFaultsDescriptor = createMetricPageFaultsDescriptor()

func createMetricPageFaultsDescriptor() pdata.MetricDescriptor {
	descriptor := pdata.NewMetricDescriptor()
	descriptor.InitEmpty()
	descriptor.SetName("host/swap/page_faults")
	descriptor.SetDescription("The number of page faults.")
	descriptor.SetUnit("1")
	descriptor.SetType(pdata.MetricTypeMonotonicInt64)
	return descriptor
}
