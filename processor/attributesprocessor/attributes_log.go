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

package attributesprocessor

import (
	"context"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/processor/filterlog"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

type logAttributesProcessor struct {
	attrProc *processorhelper.AttrProc
	include  filterlog.Matcher
	exclude  filterlog.Matcher
}

// newLogAttributesProcessor returns a processor that modifies attributes of a
// log record. To construct the attributes processors, the use of the factory
// methods are required in order to validate the inputs.
func newLogAttributesProcessor(attrProc *processorhelper.AttrProc, include, exclude filterlog.Matcher) *logAttributesProcessor {
	return &logAttributesProcessor{
		attrProc: attrProc,
		include:  include,
		exclude:  exclude,
	}
}

// ProcessLogs implements the LogsProcessor
func (a *logAttributesProcessor) ProcessLogs(_ context.Context, ld pdata.Logs) (pdata.Logs, error) {
	rls := ld.ResourceLogs()
	for i := 0; i < rls.Len(); i++ {
		rs := rls.At(i)
		if rs.IsNil() {
			continue
		}
		ilss := rls.At(i).InstrumentationLibraryLogs()
		for j := 0; j < ilss.Len(); j++ {
			ils := ilss.At(j)
			if ils.IsNil() {
				continue
			}
			logs := ils.Logs()
			for k := 0; k < logs.Len(); k++ {
				lr := logs.At(k)
				if lr.IsNil() {
					// Do not create empty log records just to add attributes
					continue
				}

				if a.skipLog(lr) {
					continue
				}

				a.attrProc.Process(lr.Attributes())
			}
		}
	}
	return ld, nil
}

// skipLog determines if a log should be processed.
// True is returned when a log should be skipped.
// False is returned when a log should not be skipped.
// The logic determining if a log should be processed is set
// in the attribute configuration with the include and exclude settings.
// Include properties are checked before exclude settings are checked.
func (a *logAttributesProcessor) skipLog(lr pdata.LogRecord) bool {
	if a.include != nil {
		// A false returned in this case means the log should not be processed.
		if include := a.include.MatchLogRecord(lr); !include {
			return true
		}
	}

	if a.exclude != nil {
		// A true returned in this case means the log should not be processed.
		if exclude := a.exclude.MatchLogRecord(lr); exclude {
			return true
		}
	}

	return false
}
