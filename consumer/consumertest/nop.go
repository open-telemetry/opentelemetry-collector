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

var (
	nopInstance = &nopConsumer{}
)

type nopConsumer struct{}

func (nc *nopConsumer) ConsumeTraces(context.Context, pdata.Traces) error {
	return nil
}

func (nc *nopConsumer) ConsumeMetrics(context.Context, pdata.Metrics) error {
	return nil
}

func (nc *nopConsumer) ConsumeLogs(context.Context, pdata.Logs) error {
	return nil
}

// NewTracesNop returns a consumer.Traces that just drops all received data and returns no error.
func NewTracesNop() consumer.Traces {
	return nopInstance
}

// NewMetricsNop returns a consumer.Metrics that just drops all received data and returns no error.
func NewMetricsNop() consumer.Metrics {
	return nopInstance
}

// NewLogsNop returns a consumer.Logs that just drops all received data and returns no error.
func NewLogsNop() consumer.Logs {
	return nopInstance
}
