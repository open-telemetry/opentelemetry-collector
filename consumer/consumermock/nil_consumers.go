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

package consumermock

import (
	"context"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumerdata"
	"go.opentelemetry.io/collector/consumer/pdata"
)

// Nil implements all consumer interfaces but drops all incoming data.
var Nil = &nilConsumer{}

var _ consumer.TraceConsumer = (*nilConsumer)(nil)
var _ consumer.TraceConsumerOld = (*nilConsumer)(nil)
var _ consumer.MetricsConsumer = (*nilConsumer)(nil)
var _ consumer.MetricsConsumerOld = (*nilConsumer)(nil)

type nilConsumer struct {
}

func (n nilConsumer) ConsumeMetricsData(ctx context.Context, md consumerdata.MetricsData) error {
	return nil
}

func (n nilConsumer) ConsumeMetrics(ctx context.Context, md pdata.Metrics) error {
	return nil
}

func (n nilConsumer) ConsumeTraceData(ctx context.Context, td consumerdata.TraceData) error {
	return nil
}

func (n nilConsumer) ConsumeTraces(ctx context.Context, td pdata.Traces) error {
	return nil
}

