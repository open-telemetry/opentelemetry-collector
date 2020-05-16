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

package filesystemscraper

import (
	"go.opentelemetry.io/collector/consumer/pdata"
)

// filesystem metric constants

const (
	deviceLabelName = "device"
	stateLabelName  = "state"
)

const (
	freeLabelValue     = "free"
	usedLabelValue     = "used"
	reservedLabelValue = "reserved"
)

var metricFilesystemUsedDescriptor = createMetricFilesystemUsedDescriptor()

func createMetricFilesystemUsedDescriptor() pdata.MetricDescriptor {
	descriptor := pdata.NewMetricDescriptor()
	descriptor.InitEmpty()
	descriptor.SetName("host/filesystem/used")
	descriptor.SetDescription("Filesystem bytes used.")
	descriptor.SetUnit("bytes")
	descriptor.SetType(pdata.MetricTypeGaugeInt64)
	return descriptor
}

var metricFilesystemINodesUsedDescriptor = createMetricFilesystemINodesUsedDescriptor()

func createMetricFilesystemINodesUsedDescriptor() pdata.MetricDescriptor {
	descriptor := pdata.NewMetricDescriptor()
	descriptor.InitEmpty()
	descriptor.SetName("host/filesystem/inodes/used")
	descriptor.SetDescription("FileSystem operations count.")
	descriptor.SetUnit("1")
	descriptor.SetType(pdata.MetricTypeGaugeInt64)
	return descriptor
}
