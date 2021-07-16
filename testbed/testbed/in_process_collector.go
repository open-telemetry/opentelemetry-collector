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
	"fmt"
	"strings"

	"github.com/shirou/gopsutil/process"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/internal/version"
	"go.opentelemetry.io/collector/service"
	"go.opentelemetry.io/collector/service/parserprovider"
)

// inProcessCollector implements the OtelcolRunner interfaces running a single otelcol as a go routine within the
// same process as the test executor.
type inProcessCollector struct {
	logger    *zap.Logger
	factories component.Factories
	configStr string
	svc       *service.Collector
	appDone   chan struct{}
	stopped   bool
}

// NewInProcessCollector crewtes a new inProcessCollector using the supplied component factories.
func NewInProcessCollector(factories component.Factories) OtelcolRunner {
	return &inProcessCollector{
		factories: factories,
	}
}

func (ipp *inProcessCollector) PrepareConfig(configStr string) (configCleanup func(), err error) {
	configCleanup = func() {
		// NoOp
	}
	var logger *zap.Logger
	logger, err = configureLogger()
	if err != nil {
		return configCleanup, err
	}
	ipp.logger = logger
	ipp.configStr = configStr
	return configCleanup, err
}

func (ipp *inProcessCollector) Start(args StartParams) error {
	settings := service.CollectorSettings{
		BuildInfo: component.BuildInfo{
			Command: "otelcol",
			Version: version.Version,
		},
		Factories:      ipp.factories,
		ParserProvider: parserprovider.NewInMemory(strings.NewReader(ipp.configStr)),
	}
	var err error
	ipp.svc, err = service.New(settings)
	if err != nil {
		return err
	}
	ipp.svc.Command().SetArgs(args.CmdArgs)

	ipp.appDone = make(chan struct{})
	go func() {
		defer close(ipp.appDone)
		appErr := ipp.svc.Run()
		if appErr != nil {
			err = appErr
		}
	}()

	for state := range ipp.svc.GetStateChannel() {
		switch state {
		case service.Starting:
			// NoOp
		case service.Running:
			return err
		default:
			err = fmt.Errorf("unable to start, otelcol state is %d", state)
		}
	}
	return err
}

func (ipp *inProcessCollector) Stop() (stopped bool, err error) {
	if !ipp.stopped {
		ipp.stopped = true
		ipp.svc.Shutdown()
	}
	<-ipp.appDone
	stopped = ipp.stopped
	return stopped, err
}

func (ipp *inProcessCollector) WatchResourceConsumption() error {
	return nil
}

func (ipp *inProcessCollector) GetProcessMon() *process.Process {
	return nil
}

func (ipp *inProcessCollector) GetTotalConsumption() *ResourceConsumption {
	return &ResourceConsumption{
		CPUPercentAvg: 0,
		CPUPercentMax: 0,
		RAMMiBAvg:     0,
		RAMMiBMax:     0,
	}
}

func (ipp *inProcessCollector) GetResourceConsumption() string {
	return ""
}

func configureLogger() (*zap.Logger, error) {
	conf := zap.NewDevelopmentConfig()
	conf.Level.SetLevel(zapcore.InfoLevel)
	logger, err := conf.Build()
	return logger, err
}
