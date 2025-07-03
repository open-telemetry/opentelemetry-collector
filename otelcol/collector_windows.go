// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build windows

package otelcol // import "go.opentelemetry.io/collector/otelcol"

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"

	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/service"
)

type windowsService struct {
	settings CollectorSettings
	col      *Collector
	flags    *flag.FlagSet
}

type collectorState struct {
	state State
	err   error
}

// NewSvcHandler constructs a new svc.Handler using the given CollectorSettings.
func NewSvcHandler(set CollectorSettings) svc.Handler {
	return &windowsService{settings: set, flags: flags(featuregate.GlobalRegistry())}
}

// Execute implements https://godoc.org/golang.org/x/sys/windows/svc#Handler
func (s *windowsService) Execute(args []string, requests <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	// The first argument supplied to service.Execute is the service name. If this is
	// not provided for some reason, raise a relevant error to the system event log
	if len(args) == 0 {
		return false, uint32(windows.ERROR_INVALID_SERVICENAME)
	}

	elog, err := openEventLog(args[0])
	if err != nil {
		return false, uint32(windows.ERROR_EVENTLOG_CANT_START)
	}

	// Start the collector in the background and then begin responding to service control requests.
	// The state channel will report if it reaches a running state and/or finishes execution (possibly
	// with an error).
	// NOTE: The `services.msc` GUI disables stop option for SERVICE_START_PENDING, but it is still
	//		 possible to send SERVICE_CONTROL_STOP using `sc.exe` / PowerShell / Windows API.
	changes <- svc.Status{State: svc.StartPending, Accepts: svc.AcceptStop | svc.AcceptShutdown}
	colStateChannel, err := s.startAsync(elog)
	if err != nil {
		elog.Error(3, fmt.Sprintf("failed to start service: %v", err))
		return false, uint32(windows.ERROR_EXCEPTION_IN_SERVICE)
	}

	// The service is always updated with SERVICE_STOPPED by svc.Run, using the returned `errno` to
	// indicate whether it was successful or finished abnormally.
	for {
		select {
		case req := <-requests:
			switch req.Cmd {
			case svc.Interrogate:
				changes <- req.CurrentStatus

			case svc.Stop, svc.Shutdown:
				// Asynchronously begin the stop process. Once col.Run finishes, a state update will
				// be received and handled to return an appropriate result.
				changes <- svc.Status{State: svc.StopPending}
				s.col.Shutdown()

			default:
				elog.Warning(3, fmt.Sprintf("unexpected service control request #%d", req.Cmd))
				return false, uint32(windows.ERROR_CALL_NOT_IMPLEMENTED)
			}
		case colState := <-colStateChannel:
			if colState.state == StateRunning {
				// Start has completed, notify the service manager.
				changes <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}
			} else if colState.state == StateClosed {
				if colState.err == nil {
					// col.Run exited cleanly as the result of a shutdown request.
					return false, uint32(windows.NO_ERROR)
				}
				elog.Error(3, fmt.Sprintf("Fatal error: %v", colState.err))
				return false, uint32(windows.ERROR_EXCEPTION_IN_SERVICE)
			}
		}
	}
}

func (s *windowsService) startAsync(elog *eventlog.Log) (<-chan collectorState, error) {
	// Parse all the flags manually.
	if err := s.flags.Parse(os.Args[1:]); err != nil {
		return nil, err
	}

	var err error
	err = updateSettingsUsingFlags(&s.settings, s.flags)
	if err != nil {
		return nil, err
	}
	s.col, err = NewCollector(s.settings)
	if err != nil {
		return nil, err
	}

	// The logging options need to be in place before the collector Run method is called
	// since the telemetry creates the logger at the time of the Run method call.
	// However, the zap.WrapCore function needs to read the serviceConfig to determine
	// if the Windows Event Log should be used, however, the serviceConfig is also
	// only read at the time of the Run method call. To work around this, we pass the
	// serviceConfig as a pointer to the logging options, and then read its value
	// when the zap.Logger is created by the telemetry.
	s.col.set.LoggingOptions = loggingOptionsWithEventLogCore(elog, &s.col.serviceConfig, s.col.set.LoggingOptions)

	// State updates are sent back via this channel.
	// It's never explicitly closed because multiple, unsynchronized goroutines send to it
	// (one for reporting back when finished with any error and one for monitoring startup).
	colStateChannel := make(chan collectorState)

	// col.Run blocks until receiving a SIGTERM signal or fatal error, so it's executed to
	// completion in a goroutine with the return error being captured and sent to the state
	// state update channel.
	go func() {
		err := s.col.Run(context.Background())
		colStateChannel <- collectorState{
			state: StateClosed,
			err:   err,
		}
	}()

	// Monitor for the collector to reach the Running state in the background and then send
	// a state update so the service manager can transition from START_PENDING -> STARTED.
	go func() {
		for {
			state := s.col.GetState()
			switch state {
			case StateRunning:
				colStateChannel <- collectorState{state: StateRunning}
				return
			case StateClosed:
				// Collector finished running - any error will be reported by the other goroutine.
				return
			default:
				// Still starting, poll again in a bit.
				time.Sleep(time.Millisecond * 200)
			}
		}
	}()

	// Return immediately, the caller can monitor the returned channel to know when the collector has reached
	// the running state and/or stopped with an error.
	return colStateChannel, nil
}

func openEventLog(serviceName string) (*eventlog.Log, error) {
	elog, err := eventlog.Open(serviceName)
	if err != nil {
		return nil, fmt.Errorf("service failed to open event log: %w", err)
	}

	return elog, nil
}

func loggingOptionsWithEventLogCore(
	elog *eventlog.Log,
	serviceConfig **service.Config,
	userOptions []zap.Option,
) []zap.Option {
	return append(
		// The order below must be preserved - see PR #11051
		// The event log core must run *after* any user provided options, so it
		// must be the first option in this list.
		[]zap.Option{zap.WrapCore(withWindowsCore(elog, serviceConfig))},
		userOptions...,
	)
}

var _ zapcore.Core = (*windowsEventLogCore)(nil)

type windowsEventLogCore struct {
	core    zapcore.Core
	elog    *eventlog.Log
	encoder zapcore.Encoder
}

func (w windowsEventLogCore) Enabled(level zapcore.Level) bool {
	return w.core.Enabled(level)
}

func (w windowsEventLogCore) With(fields []zapcore.Field) zapcore.Core {
	enc := w.encoder.Clone()
	for _, field := range fields {
		field.AddTo(enc)
	}
	return windowsEventLogCore{
		core:    w.core,
		elog:    w.elog,
		encoder: enc,
	}
}

func (w windowsEventLogCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if w.Enabled(ent.Level) {
		return ce.AddCore(ent, w)
	}
	return ce
}

func (w windowsEventLogCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	buf, err := w.encoder.EncodeEntry(ent, fields)
	if err != nil {
		w.elog.Warning(2, fmt.Sprintf("failed encoding log entry %v\r\n", err))
		return err
	}
	msg := buf.String()
	buf.Free()

	switch ent.Level {
	case zapcore.FatalLevel, zapcore.PanicLevel, zapcore.DPanicLevel:
		// golang.org/x/sys/windows/svc/eventlog does not support Critical level event logs
		return w.elog.Error(3, msg)
	case zapcore.ErrorLevel:
		return w.elog.Error(3, msg)
	case zapcore.WarnLevel:
		return w.elog.Warning(2, msg)
	case zapcore.InfoLevel:
		return w.elog.Info(1, msg)
	}
	// We would not be here if debug were disabled so log as info to not drop.
	return w.elog.Info(1, msg)
}

func (w windowsEventLogCore) Sync() error {
	return w.core.Sync()
}

func withWindowsCore(elog *eventlog.Log, serviceConfig **service.Config) func(zapcore.Core) zapcore.Core {
	return func(core zapcore.Core) zapcore.Core {
		if serviceConfig != nil && *serviceConfig != nil {
			for _, output := range (*serviceConfig).Telemetry.Logs.OutputPaths {
				if output != "stdout" && output != "stderr" {
					// A log file was specified in the configuration, so we should not use the Windows Event Log
					return core
				}
			}
		}

		// Use the Windows Event Log
		encoderConfig := zap.NewProductionEncoderConfig()
		encoderConfig.LineEnding = "\r\n"
		encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		return windowsEventLogCore{core, elog, zapcore.NewConsoleEncoder(encoderConfig)}
	}
}
