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

package testbed

import (
	"github.com/shirou/gopsutil/process"
)

type StartParams struct {
	Name         string
	LogFilePath  string
	CmdArgs      []string
	resourceSpec *ResourceSpec
}

type ResourceConsumption struct {
	CPUPercentAvg float64
	CPUPercentMax float64
	RAMMiBAvg     uint32
	RAMMiBMax     uint32
}

// OtelcolRunner defines the interface for configuring, starting and stopping one or more instances of
// otelcol which will be the subject of testing being executed.
type OtelcolRunner interface {
	// PrepareConfig stores the provided YAML-based otelcol configuration file in the format needed by the otelcol
	// instance(s) this runner manages. If successful, it returns the cleanup config function to be executed after
	// the test is executed.
	PrepareConfig(configStr string) (configCleanup func(), err error)
	// Start starts the otelcol instance(s) if not already running which is the subject of the test to be run.
	// It returns the host:port of the data receiver to post test data to.
	Start(args StartParams) error
	// Stop stops the otelcol instance(s) which are the subject of the test just run if applicable. Returns whether
	// the instance was actually stopped or not.
	Stop() (stopped bool, err error)
	// WatchResourceConsumption toggles on the monitoring of resource consumpution by the otelcol instance under test.
	WatchResourceConsumption() error
	// GetProcessMon returns the Process being used to monitor resource consumption.
	GetProcessMon() *process.Process
	// GetTotalConsumption returns the data collected by the process monitor.
	GetTotalConsumption() *ResourceConsumption
	// GetResourceConsumption returns the data collected by the process monitor as a display string.
	GetResourceConsumption() string
}
