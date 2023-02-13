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

package fanoutconsumer // import "go.opentelemetry.io/collector/service/internal/fanoutconsumer"

import (
	"context"
	"fmt"

	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

// NewMetrics wraps multiple metrics consumers in a single one.
// It fanouts the incoming data to all the consumers, and does smart routing:
//   - Clones only to the consumer that needs to mutate the data.
//   - If all consumers needs to mutate the data one will get the original data.
func NewMetrics(mcs []consumer.Metrics) consumer.Metrics {
	if len(mcs) == 1 {
		// Don't wrap if no need to do it.
		return mcs[0]
	}
	var pass []consumer.Metrics
	var clone []consumer.Metrics
	for i := 0; i < len(mcs)-1; i++ {
		if !mcs[i].Capabilities().MutatesData {
			pass = append(pass, mcs[i])
		} else {
			clone = append(clone, mcs[i])
		}
	}
	// Give the original data to the last consumer if no other read-only consumer,
	// otherwise put it in the right bucket. Never share the same data between
	// a mutating and a non-mutating consumer since the non-mutating consumer may process
	// data async and the mutating consumer may change the data before that.
	if len(pass) == 0 || !mcs[len(mcs)-1].Capabilities().MutatesData {
		pass = append(pass, mcs[len(mcs)-1])
	} else {
		clone = append(clone, mcs[len(mcs)-1])
	}
	return &metricsConsumer{pass: pass, clone: clone}
}

type metricsConsumer struct {
	pass  []consumer.Metrics
	clone []consumer.Metrics
}

func (msc *metricsConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// ConsumeMetrics exports the pmetric.Metrics to all consumers wrapped by the current one.
func (msc *metricsConsumer) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	var errs error
	// Initially pass to clone exporter to avoid the case where the optimization of sending
	// the incoming data to a mutating consumer is used that may change the incoming data before
	// cloning.
	for _, mc := range msc.clone {
		clonedMetrics := pmetric.NewMetrics()
		md.CopyTo(clonedMetrics)
		errs = multierr.Append(errs, mc.ConsumeMetrics(ctx, clonedMetrics))
	}
	for _, mc := range msc.pass {
		errs = multierr.Append(errs, mc.ConsumeMetrics(ctx, md))
	}
	return errs
}

var _ connector.MetricsRouter = (*metricsRouter)(nil)

type metricsRouter struct {
	consumer.Metrics
	consumers map[component.ID]consumer.Metrics
}

func NewMetricsRouter(cm map[component.ID]consumer.Metrics) consumer.Metrics {
	consumers := make([]consumer.Metrics, 0, len(cm))
	for _, consumer := range cm {
		consumers = append(consumers, consumer)
	}
	return &metricsRouter{
		Metrics:   NewMetrics(consumers),
		consumers: cm,
	}
}

func (r *metricsRouter) PipelineIDs() []component.ID {
	ids := make([]component.ID, 0, len(r.consumers))
	for id := range r.consumers {
		ids = append(ids, id)
	}
	return ids
}

func (r *metricsRouter) Consumer(pipelineIDs ...component.ID) (consumer.Metrics, error) {
	if len(pipelineIDs) == 0 {
		return nil, fmt.Errorf("missing consumers")
	}
	consumers := make([]consumer.Metrics, 0, len(pipelineIDs))
	var errors error
	for _, pipelineID := range pipelineIDs {
		c, ok := r.consumers[pipelineID]
		if ok {
			consumers = append(consumers, c)
		} else {
			errors = multierr.Append(errors, fmt.Errorf("missing consumer: %q", pipelineID))
		}
	}
	if errors != nil {
		// TODO potentially this could return a NewMetrics with the valid consumers
		return nil, errors
	}
	return NewMetrics(consumers), nil
}
