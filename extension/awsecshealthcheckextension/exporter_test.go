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

package awsecshealthcheckextension

import (
	"go.opencensus.io/stats/view"
	"gotest.tools/v3/assert"
	"testing"
	"time"
)

func TestECSHealthCheckExporter_ExportView(t *testing.T) {
	exporter := &ECSHealthCheckExporter{}
	vd := &view.Data{
		View:  nil,
		Start: time.Time{},
		End:   time.Time{},
		Rows:  nil,
	}
	exporter.ExportView(vd)
	assert.Equal(t, 1, len(exporter.exporterErrorQueue))
}

func TestECSHealthCheckExporter_rotate(t *testing.T) {
	exporter := &ECSHealthCheckExporter{}
	currentTime := time.Now()
	time1 := currentTime.Add(-10 * time.Minute)
	time2 := currentTime.Add(-3 * time.Minute)
	vd1 := &view.Data{
		View:  nil,
		Start: time1,
		End:   currentTime,
		Rows:  nil,
	}
	vd2 := &view.Data{
		View:  nil,
		Start: time2,
		End:   currentTime,
		Rows:  nil,
	}
	exporter.ExportView(vd1)
	exporter.ExportView(vd2)
	exporter.rotate(5 * time.Minute)
	assert.Equal(t, 1, len(exporter.exporterErrorQueue))
}
