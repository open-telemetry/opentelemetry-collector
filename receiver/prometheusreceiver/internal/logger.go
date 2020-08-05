// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internal

import (
	"fmt"

	gokitLog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewZapToGokitLogAdapter create an adapter for zap.Logger to gokitLog.Logger
func NewZapToGokitLogAdapter(logger *zap.Logger) gokitLog.Logger {
	// need to skip two levels in order to get the correct caller
	// one for this method, the other for gokitLog
	logger = logger.WithOptions(zap.AddCallerSkip(2))
	return &zapToGokitLogAdapter{l: logger.Sugar()}
}

type zapToGokitLogAdapter struct {
	l *zap.SugaredLogger
}

func (w *zapToGokitLogAdapter) Log(keyvals ...interface{}) error {
	// expecting key value pairs, the number of items need to be even
	if len(keyvals)%2 == 0 {
		zapLevel, keyvals := extractLogLevel(keyvals)
		logFunc, err := levelToFunc(w.l, zapLevel)
		if err != nil {
			return err
		}
		logFunc("", keyvals...)
	} else {
		// in case something goes wrong
		w.l.Info(keyvals...)
	}
	return nil
}

// extract go-kit log level from key value pairs, convert it to zap log level
// and remove from the list to avoid duplication in log message
func extractLogLevel(keyvals []interface{}) (zapcore.Level, []interface{}) {
	zapLevel := zapcore.InfoLevel
	output := make([]interface{}, 0, len(keyvals))
	for i := 0; i < len(keyvals); i = i + 2 {
		key := keyvals[i]
		val := keyvals[i+1]

		if l, ok := matchLogLevel(key, val); ok {
			zapLevel = *l
			continue
		}

		output = append(output, key, val)
	}

	return zapLevel, output
}

// check if a given key-value pair represents go-kit log level and return
// a corresponding zap log level
func matchLogLevel(key interface{}, val interface{}) (*zapcore.Level, bool) {
	strKey, ok := key.(string)
	if !ok || strKey != "level" {
		return nil, false
	}

	levelVal, ok := val.(level.Value)
	if !ok {
		return nil, false
	}

	zapLevel := toZapLevel(levelVal)
	return &zapLevel, true
}

// convert go-kit log level to zap
func toZapLevel(value level.Value) zapcore.Level {
	// See https://github.com/go-kit/kit/blob/556100560949062d23fe05d9fda4ce173c30c59f/log/level/level.go#L184-L187
	switch value.String() {
	case "error":
		return zapcore.ErrorLevel
	case "warn":
		return zapcore.WarnLevel
	case "info":
		return zapcore.InfoLevel
	case "debug":
		return zapcore.DebugLevel
	default:
		return zapcore.InfoLevel
	}
}

// find a matching zap logging function to be used for a given level
func levelToFunc(logger *zap.SugaredLogger, lvl zapcore.Level) (func(string, ...interface{}), error) {
	switch lvl {
	case zapcore.DebugLevel:
		return logger.Debugf, nil
	case zapcore.InfoLevel:
		return logger.Infof, nil
	case zapcore.WarnLevel:
		return logger.Warnf, nil
	case zapcore.ErrorLevel:
		return logger.Errorf, nil
	case zapcore.DPanicLevel:
		return logger.DPanicf, nil
	case zapcore.PanicLevel:
		return logger.Panicf, nil
	case zapcore.FatalLevel:
		return logger.Fatalf, nil
	}
	return nil, fmt.Errorf("unrecognized level: %q", lvl)
}

var _ gokitLog.Logger = (*zapToGokitLogAdapter)(nil)
