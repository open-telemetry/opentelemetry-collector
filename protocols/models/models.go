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

package models

import (
	"fmt"
	"go.opentelemetry.io/collector/consumer/pdata"
)

type MetricsModelTranslator interface {
	// MetricsFromModel converts a data model of another protocol into pdata.
	MetricsFromModel(src interface{}) (pdata.Metrics, error)
	// MetricsToModel converts pdata to data model.
	MetricsToModel(md pdata.Metrics, out interface{}) error
}

type TracesModelTranslator interface {
	// TracesFromModel converts a data model of another protocol into pdata.
	TracesFromModel(src interface{}) (pdata.Traces, error)
	// TracesToModel converts pdata to data model.
	TracesToModel(md pdata.Traces, out interface{}) error
}

type LogsModelTranslator interface {
	// LogsFromModel converts a data model of another protocol into pdata.
	LogsFromModel(src interface{}) (pdata.Logs, error)
	// LogsToModel converts pdata to data model.
	LogsToModel(md pdata.Logs, out interface{}) error
}

// ErrIncompatibleType details a type conversion error during translation.
type ErrIncompatibleType struct {
	Model interface{}
	// TODO: maybe do expected vs. actual?
}

func (i *ErrIncompatibleType) Error() string {
	return fmt.Sprintf("model type %T is incompatible", i.Model)
}
