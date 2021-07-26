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

package diskscraper

import (
	"go.opentelemetry.io/collector/model/pdata"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/metadata"
)

func initializeNumberDataPointAsInt(dataPoint pdata.NumberDataPoint, startTime, now pdata.Timestamp, deviceLabel string, directionLabel string, value int64) {
	labelsMap := dataPoint.LabelsMap()
	labelsMap.Insert(metadata.Labels.DiskDevice, deviceLabel)
	if directionLabel != "" {
		labelsMap.Insert(metadata.Labels.DiskDirection, directionLabel)
	}
	dataPoint.SetStartTimestamp(startTime)
	dataPoint.SetTimestamp(now)
	dataPoint.SetIntVal(value)
}

func initializeNumberDataPointAsDouble(dataPoint pdata.NumberDataPoint, startTime, now pdata.Timestamp, deviceLabel string, directionLabel string, value float64) {
	labelsMap := dataPoint.LabelsMap()
	labelsMap.Insert(metadata.Labels.DiskDevice, deviceLabel)
	if directionLabel != "" {
		labelsMap.Insert(metadata.Labels.DiskDirection, directionLabel)
	}
	dataPoint.SetStartTimestamp(startTime)
	dataPoint.SetTimestamp(now)
	dataPoint.SetDoubleVal(value)
}
