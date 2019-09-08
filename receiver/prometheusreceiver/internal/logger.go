// Copyright 2019, OpenTelemetry Authors
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
	gokitLog "github.com/go-kit/kit/log"
	"go.uber.org/zap"
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
	if len(keyvals)%2 == 0 {
		// expecting key value pairs, the number of items need to be even
		w.l.Infow("", keyvals...)
	} else {
		// in case something goes wrong
		w.l.Info(keyvals...)
	}
	return nil
}

var _ gokitLog.Logger = (*zapToGokitLogAdapter)(nil)
