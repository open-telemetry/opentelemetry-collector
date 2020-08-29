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

package filesystemscraper

import (
	"go.opentelemetry.io/collector/internal/dataold"
)

// labels

const (
	deviceLabelName = "device"
	stateLabelName  = "state"
)

// state label values

const (
	freeLabelValue     = "free"
	reservedLabelValue = "reserved"
	usedLabelValue     = "used"
)

// descriptors

var fileSystemUsageDescriptor = func() dataold.MetricDescriptor {
	descriptor := dataold.NewMetricDescriptor()
	descriptor.InitEmpty()
	descriptor.SetName("system.filesystem.usage")
	descriptor.SetDescription("Filesystem bytes used.")
	descriptor.SetUnit("bytes")
	descriptor.SetType(dataold.MetricTypeInt64)
	return descriptor
}()

var fileSystemINodesUsageDescriptor = func() dataold.MetricDescriptor {
	descriptor := dataold.NewMetricDescriptor()
	descriptor.InitEmpty()
	descriptor.SetName("system.filesystem.inodes.usage")
	descriptor.SetDescription("FileSystem operations count.")
	descriptor.SetUnit("1")
	descriptor.SetType(dataold.MetricTypeInt64)
	return descriptor
}()
