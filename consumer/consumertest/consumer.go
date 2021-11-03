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

package consumertest // import "go.opentelemetry.io/collector/consumer/consumertest"

import (
	"context"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/model/pdata"
)

// Consumer is a convenience interface that implements all consumer interfaces.
// It has a private function on it to forbid external users from implementing it
// and, as a result, to allow us to add extra functions without breaking
// compatibility.
type Consumer interface {
	// Capabilities to implement the base consumer functionality.
	Capabilities() consumer.Capabilities
	// ConsumeTraces to implement the consumer.Traces.
	ConsumeTraces(context.Context, pdata.Traces) error
	// ConsumeMetrics to implement the consumer.Metrics.
	ConsumeMetrics(context.Context, pdata.Metrics) error
	// ConsumeLogs to implement the consumer.Logs.
	ConsumeLogs(context.Context, pdata.Logs) error
	unexported()
}

var _ consumer.Logs = (Consumer)(nil)
var _ consumer.Metrics = (Consumer)(nil)
var _ consumer.Traces = (Consumer)(nil)
