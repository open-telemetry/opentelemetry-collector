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

package kafkareceiver

import (
	"go.opentelemetry.io/collector/internal/model"
)

type pdataLogsUnmarshaler struct {
	model.LogsUnmarshaler
	encoding string
}

func (p pdataLogsUnmarshaler) Encoding() string {
	return p.encoding
}

func newPdataLogsUnmarshaler(unmarshaler model.LogsUnmarshaler, encoding string) LogsUnmarshaler {
	return pdataLogsUnmarshaler{
		LogsUnmarshaler: unmarshaler,
		encoding:        encoding,
	}
}

type pdataTracesUnmarshaler struct {
	model.TracesUnmarshaler
	encoding string
}

func (p pdataTracesUnmarshaler) Encoding() string {
	return p.encoding
}

func newPdataTracesUnmarshaler(unmarshaler model.TracesUnmarshaler, encoding string) TracesUnmarshaler {
	return pdataTracesUnmarshaler{
		TracesUnmarshaler: unmarshaler,
		encoding:          encoding,
	}
}

type pdataMetricsUnmarshaler struct {
	model.MetricsUnmarshaler
	encoding string
}

func (p pdataMetricsUnmarshaler) Encoding() string {
	return p.encoding
}

func newPdataMetricsUnmarshaler(unmarshaler model.MetricsUnmarshaler, encoding string) MetricsUnmarshaler {
	return pdataMetricsUnmarshaler{
		MetricsUnmarshaler: unmarshaler,
		encoding:           encoding,
	}
}
