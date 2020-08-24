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

package internal

import (
	gokitLog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"go.uber.org/zap"
)

const (
	levelKey = "level"
	msgKey   = "msg"
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

type logData struct {
	level       level.Value
	msg         string
	otherFields []interface{}
}

func (w *zapToGokitLogAdapter) Log(keyvals ...interface{}) error {
	// expecting key value pairs, the number of items need to be even
	if len(keyvals)%2 == 0 {
		// Extract log level and message and log them using corresponding zap function
		ld := extractLogData(keyvals)
		logFunc := levelToFunc(w.l, ld.level)
		logFunc(ld.msg, ld.otherFields...)
	} else {
		// in case something goes wrong
		w.l.Info(keyvals...)
	}
	return nil
}

func extractLogData(keyvals []interface{}) *logData {
	lvl := level.InfoValue() // default
	msg := ""

	other := make([]interface{}, 0, len(keyvals))
	for i := 0; i < len(keyvals); i += 2 {
		key := keyvals[i]
		val := keyvals[i+1]

		if l, ok := matchLogLevel(key, val); ok {
			lvl = l
			continue
		}

		if m, ok := matchLogMessage(key, val); ok {
			msg = m
			continue
		}

		other = append(other, key, val)
	}

	return &logData{
		level:       lvl,
		msg:         msg,
		otherFields: other,
	}
}

// check if a given key-value pair represents go-kit log message and return it
func matchLogMessage(key interface{}, val interface{}) (string, bool) {
	strKey, ok := key.(string)
	if !ok || strKey != msgKey {
		return "", false
	}

	msg, ok := val.(string)
	if !ok {
		return "", false
	}

	return msg, true
}

// check if a given key-value pair represents go-kit log level and return it
func matchLogLevel(key interface{}, val interface{}) (level.Value, bool) {
	strKey, ok := key.(string)
	if !ok || strKey != levelKey {
		return nil, false
	}

	levelVal, ok := val.(level.Value)
	if !ok {
		return nil, false
	}

	return levelVal, true
}

// find a matching zap logging function to be used for a given level
func levelToFunc(logger *zap.SugaredLogger, lvl level.Value) func(string, ...interface{}) {
	switch lvl {
	case level.DebugValue():
		return logger.Debugw
	case level.InfoValue():
		return logger.Infow
	case level.WarnValue():
		return logger.Warnw
	case level.ErrorValue():
		return logger.Errorw
	}

	// default
	return logger.Infof
}

var _ gokitLog.Logger = (*zapToGokitLogAdapter)(nil)
