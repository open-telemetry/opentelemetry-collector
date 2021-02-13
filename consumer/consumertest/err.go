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

package consumertest

import (
	"context"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/pdata"
)

type errConsumer struct {
	err error
}

func (er *errConsumer) ConsumeTraces(context.Context, pdata.Traces) error {
	return er.err
}

func (er *errConsumer) ConsumeMetrics(context.Context, pdata.Metrics) error {
	return er.err
}

func (er *errConsumer) ConsumeLogs(context.Context, pdata.Logs) error {
	return er.err
}

// NewTracesErr returns a consumer.TracesConsumer that just drops all received data and returns the given error.
func NewTracesErr(err error) consumer.TracesConsumer {
	return &errConsumer{err: err}
}

// NewMetricsErr returns a consumer.MetricsConsumer that just drops all received data and returns the given error.
func NewMetricsErr(err error) consumer.MetricsConsumer {
	return &errConsumer{err: err}
}

// NewLogsErr returns a consumer.LogsConsumer that just drops all received data and returns the given error.
func NewLogsErr(err error) consumer.LogsConsumer {
	return &errConsumer{err: err}
}
