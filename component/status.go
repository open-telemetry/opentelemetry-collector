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

package component // import "go.opentelemetry.io/collector/component"

import "go.opentelemetry.io/collector/config"

type StatusReport struct {
	ComponentID config.ComponentID
	Message     string
	Error       error
}

type StatusReportFunc func(status StatusReport)

type StatusReporters struct {
	reporters    []StatusReportFunc
	registerChan chan (StatusReportFunc)
	statusChan   chan (StatusReport)
	stopChan     chan (struct{})
}

func NewStatusReporters() *StatusReporters {
	return &StatusReporters{
		reporters:    []StatusReportFunc{},
		registerChan: make(chan StatusReportFunc),
		statusChan:   make(chan StatusReport),
		stopChan:     make(chan struct{}),
	}
}

func (srs *StatusReporters) Register(reporter StatusReportFunc) {
	srs.registerChan <- reporter
}

func (srs *StatusReporters) Report(status StatusReport) {
	srs.statusChan <- status
}

func (srs *StatusReporters) Start() {
	go func() {
		for {
			select {
			case reporter := <-srs.registerChan:
				srs.reporters = append(srs.reporters, reporter)
			case status := <-srs.statusChan:
				srs.notify(status)
			case <-srs.stopChan:
				return
			}
		}
	}()
}

func (srs *StatusReporters) Stop() {
	srs.stopChan <- struct{}{}
}

func (srs *StatusReporters) notify(status StatusReport) {
	for _, fn := range srs.reporters {
		fn(status)
	}
}
