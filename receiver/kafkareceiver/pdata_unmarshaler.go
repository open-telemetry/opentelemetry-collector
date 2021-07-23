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
	"go.opentelemetry.io/collector/model/pdata"
)

type pdataLogsUnmarshaler struct {
	pdata.LogsUnmarshaler
	encoding string
}

func (p pdataLogsUnmarshaler) Unmarshal(buf []byte) (pdata.Logs, error) {
	return p.LogsUnmarshaler.UnmarshalLogs(buf)
}

func (p pdataLogsUnmarshaler) Encoding() string {
	return p.encoding
}

func newPdataLogsUnmarshaler(unmarshaler pdata.LogsUnmarshaler, encoding string) LogsUnmarshaler {
	return pdataLogsUnmarshaler{
		LogsUnmarshaler: unmarshaler,
		encoding:        encoding,
	}
}

type pdataTracesUnmarshaler struct {
	pdata.TracesUnmarshaler
	encoding string
}

func (p pdataTracesUnmarshaler) Unmarshal(buf []byte) (pdata.Traces, error) {
	return p.TracesUnmarshaler.UnmarshalTraces(buf)
}

func (p pdataTracesUnmarshaler) Encoding() string {
	return p.encoding
}

func newPdataTracesUnmarshaler(unmarshaler pdata.TracesUnmarshaler, encoding string) TracesUnmarshaler {
	return pdataTracesUnmarshaler{
		TracesUnmarshaler: unmarshaler,
		encoding:          encoding,
	}
}

type pdataMetricsUnmarshaler struct {
	pdata.MetricsUnmarshaler
	encoding string
}

func (p pdataMetricsUnmarshaler) Unmarshal(buf []byte) (pdata.Metrics, error) {
	return p.MetricsUnmarshaler.UnmarshalMetrics(buf)
}

func (p pdataMetricsUnmarshaler) Encoding() string {
	return p.encoding
}

func newPdataMetricsUnmarshaler(unmarshaler pdata.MetricsUnmarshaler, encoding string) MetricsUnmarshaler {
	return pdataMetricsUnmarshaler{
		MetricsUnmarshaler: unmarshaler,
		encoding:           encoding,
	}
}
