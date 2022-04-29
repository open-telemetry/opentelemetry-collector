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

//go:build windows
// +build windows

package service // import "go.opentelemetry.io/collector/service"

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"

	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/mapconverter/overwritepropertiesmapconverter"
	"go.opentelemetry.io/collector/service/featuregate"
)

type windowsService struct {
	settings CollectorSettings
	col      *Collector
}

// NewSvcHandler constructs a new svc.Handler using the given CollectorSettings.
func NewSvcHandler(set CollectorSettings) svc.Handler {
	return &windowsService{settings: set}
}

// Execute implements https://godoc.org/golang.org/x/sys/windows/svc#Handler
func (s *windowsService) Execute(args []string, requests <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	// The first argument supplied to service.Execute is the service name. If this is
	// not provided for some reason, raise a relevant error to the system event log
	if len(args) == 0 {
		return false, 1213 // 1213: ERROR_INVALID_SERVICENAME
	}

	elog, err := openEventLog(args[0])
	if err != nil {
		return false, 1501 // 1501: ERROR_EVENTLOG_CANT_START
	}

	colErrorChannel := make(chan error, 1)

	changes <- svc.Status{State: svc.StartPending}
	if err = s.start(elog, colErrorChannel); err != nil {
		elog.Error(3, fmt.Sprintf("failed to start service: %v", err))
		return false, 1064 // 1064: ERROR_EXCEPTION_IN_SERVICE
	}
	changes <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}

	for req := range requests {
		switch req.Cmd {
		case svc.Interrogate:
			changes <- req.CurrentStatus

		case svc.Stop, svc.Shutdown:
			changes <- svc.Status{State: svc.StopPending}
			if err = s.stop(colErrorChannel); err != nil {
				elog.Error(3, fmt.Sprintf("errors occurred while shutting down the service: %v", err))
			}
			changes <- svc.Status{State: svc.Stopped}
			return false, 0

		default:
			elog.Error(3, fmt.Sprintf("unexpected service control request #%d", req.Cmd))
			return false, 1052 // 1052: ERROR_INVALID_SERVICE_CONTROL
		}
	}

	return false, 0
}

func (s *windowsService) start(elog *eventlog.Log, colErrorChannel chan error) error {
	// Parse all the flags manually.
	if err := flags().Parse(os.Args[1:]); err != nil {
		return err
	}
	featuregate.GetRegistry().Apply(gatesList)
	var err error
	s.col, err = newWithWindowsEventLogCore(s.settings, elog)
	if err != nil {
		return err
	}

	// col.Run blocks until receiving a SIGTERM signal, so needs to be started
	// asynchronously, but it will exit early if an error occurs on startup
	go func() {
		colErrorChannel <- s.col.Run(context.Background())
	}()

	// wait until the collector server is in the Running state
	go func() {
		for {
			state := s.col.GetState()
			if state == Running {
				colErrorChannel <- nil
				break
			}
			time.Sleep(time.Millisecond * 200)
		}
	}()

	// wait until the collector server is in the Running state, or an error was returned
	return <-colErrorChannel
}

func (s *windowsService) stop(colErrorChannel chan error) error {
	// simulate a SIGTERM signal to terminate the collector server
	s.col.signalsChannel <- syscall.SIGTERM
	// return the response of col.Start
	return <-colErrorChannel
}

func openEventLog(serviceName string) (*eventlog.Log, error) {
	elog, err := eventlog.Open(serviceName)
	if err != nil {
		return nil, fmt.Errorf("service failed to open event log: %w", err)
	}

	return elog, nil
}

func newWithWindowsEventLogCore(set CollectorSettings, elog *eventlog.Log) (*Collector, error) {
	if set.ConfigProvider == nil {
		var err error
		cfgSet := newDefaultConfigProviderSettings(getConfigFlag())
		// Append the "overwrite properties converter" as the first converter.
		cfgSet.MapConverters = append(
			[]config.MapConverterFunc{overwritepropertiesmapconverter.New(getSetFlag())},
			cfgSet.MapConverters...)
		set.ConfigProvider, err = NewConfigProvider(cfgSet)
		if err != nil {
			return nil, err
		}
	}
	if set.TelemetryProvider == nil {
		set.TelemetryProvider = newWindowsServiceTelemetryProvider(elog)
	}
	return New(set)
}
