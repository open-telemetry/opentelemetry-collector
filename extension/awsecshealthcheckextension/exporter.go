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
	"sync"
	"time"

	"go.opencensus.io/stats/view"
)

// ECSHealthCheckExporter is a struct implement the exporter interface in open census that could export metrics
type ECSHealthCheckExporter struct {
	mu                 sync.Mutex
	exporterErrorQueue []*view.Data
}

func newECSHealthCheckExporter() *ECSHealthCheckExporter {
	return &ECSHealthCheckExporter{}
}

// ExportView function could export the view to the queue
func (e *ECSHealthCheckExporter) ExportView(vd *view.Data) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.exporterErrorQueue = append(e.exporterErrorQueue, vd)
}

// rotate function could rotate the error logs that expired the time interval
func (e *ECSHealthCheckExporter) rotate(interval time.Duration) {
	viewNum := len(e.exporterErrorQueue)
	currentTime := time.Now()
	for i := 0; i < viewNum; i++ {
		vd := e.exporterErrorQueue[0]
		if vd.Start.Add(interval).After(currentTime) {
			e.exporterErrorQueue = append(e.exporterErrorQueue, vd)
		}
		e.exporterErrorQueue = e.exporterErrorQueue[1:]
	}
}
