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

package componenttest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/consumer/pdata"
)

func TestExampleExporterConsumer(t *testing.T) {
	exp := &ExampleExporterConsumer{}
	host := NewNopHost()
	assert.Equal(t, false, exp.ExporterStarted)
	err := exp.Start(context.Background(), host)
	assert.NoError(t, err)
	assert.Equal(t, true, exp.ExporterStarted)

	assert.Equal(t, 0, len(exp.Traces))
	err = exp.ConsumeTraces(context.Background(), pdata.Traces{})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(exp.Traces))

	assert.Equal(t, 0, len(exp.Metrics))
	err = exp.ConsumeMetrics(context.Background(), pdata.Metrics{})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(exp.Metrics))

	assert.Equal(t, false, exp.ExporterShutdown)
	err = exp.Shutdown(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, true, exp.ExporterShutdown)
}

func TestExampleReceiverProducer(t *testing.T) {
	rcv := &ExampleReceiverProducer{}
	host := NewNopHost()
	assert.Equal(t, false, rcv.Started)
	err := rcv.Start(context.Background(), host)
	assert.NoError(t, err)
	assert.Equal(t, true, rcv.Started)

	err = rcv.Shutdown(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, true, rcv.Started)
}
