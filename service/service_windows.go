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

// +build windows

package service

import (
	"fmt"
	"syscall"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
)

type WindowsService struct {
	params Parameters
	app    *Application
}

func NewWindowsService(params Parameters) *WindowsService {
	return &WindowsService{params: params}
}

// Execute implements https://godoc.org/golang.org/x/sys/windows/svc#Handler
func (s *WindowsService) Execute(args []string, requests <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	// The first argument supplied to service.Execute is the service name. If this is
	// not provided for some reason, raise a relevant error to the system event log
	if len(args) == 0 {
		return false, 1213 // 1213: ERROR_INVALID_SERVICENAME
	}

	elog, err := openEventLog(args[0])
	if err != nil {
		return false, 1501 // 1501: ERROR_EVENTLOG_CANT_START
	}

	appErrorChannel := make(chan error, 1)

	changes <- svc.Status{State: svc.StartPending}
	if err = s.start(elog, appErrorChannel); err != nil {
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
			if err := s.stop(appErrorChannel); err != nil {
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

func (s *WindowsService) start(elog *eventlog.Log, appErrorChannel chan error) error {
	var err error
	s.app, err = newWithEventViewerLoggingHook(s.params, elog)
	if err != nil {
		return err
	}

	// app.Start blocks until receiving a SIGTERM signal, so needs to be started
	// asynchronously, but it will exit early if an error occurs on startup
	go func() { appErrorChannel <- s.app.Run() }()

	// wait until the app is in the Running state
	go func() {
		for state := range s.app.GetStateChannel() {
			if state == Running {
				appErrorChannel <- nil
				break
			}
		}
	}()

	// wait until the app is in the Running state, or an error was returned
	return <-appErrorChannel
}

func (s *WindowsService) stop(appErrorChannel chan error) error {
	// simulate a SIGTERM signal to terminate the application
	s.app.signalsChannel <- syscall.SIGTERM
	// return the response of app.Start
	return <-appErrorChannel
}

func openEventLog(serviceName string) (*eventlog.Log, error) {
	elog, err := eventlog.Open(serviceName)
	if err != nil {
		return nil, fmt.Errorf("service failed to open event log: %w", err)
	}

	return elog, nil
}

func newWithEventViewerLoggingHook(params Parameters, elog *eventlog.Log) (*Application, error) {
	params.LoggingOptions = append(
		params.LoggingOptions,
		zap.Hooks(func(entry zapcore.Entry) error {
			msg := fmt.Sprintf("%v\r\n\r\nStack Trace:\r\n%v", entry.Message, entry.Stack)

			switch entry.Level {
			case zapcore.FatalLevel, zapcore.PanicLevel, zapcore.DPanicLevel:
				// golang.org/x/sys/windows/svc/eventlog does not support Critical level event logs
				return elog.Error(3, msg)
			case zapcore.ErrorLevel:
				return elog.Error(3, msg)
			case zapcore.WarnLevel:
				return elog.Warning(2, msg)
			case zapcore.InfoLevel:
				return elog.Info(1, msg)
			}

			// ignore Debug level logs
			return nil
		}),
	)

	return New(params)
}
