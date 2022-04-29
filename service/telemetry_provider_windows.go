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
	"fmt"

	"go.opentelemetry.io/contrib/zpages"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/sys/windows/svc/eventlog"
)

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

func withWindowsCore(elog *eventlog.Log) func(zapcore.Core) zapcore.Core {
	return func(core zapcore.Core) zapcore.Core {
		encoderConfig := zap.NewProductionEncoderConfig()
		encoderConfig.LineEnding = "\r\n"
		return windowsEventLogCore{core, elog, zapcore.NewConsoleEncoder(encoderConfig)}
	}
}

func newWindowsServiceTelemetryProvider(elog *eventlog.Log) TelemetryProvider {
	return &defaultTelemetryFactory{
		zPagesSpanProcessor: zpages.NewSpanProcessor(),
		loggingOptions:      []zap.Option{zap.WrapCore(withWindowsCore(elog))},
	}
}
