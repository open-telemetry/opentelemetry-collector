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

package testcomponents

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/model/pdata"
)

func TestExampleExporterConsumer(t *testing.T) {
	exp := &ExampleExporterConsumer{}
	host := componenttest.NewNopHost()
	assert.False(t, exp.ExporterStarted)
	err := exp.Start(context.Background(), host)
	assert.NoError(t, err)
	assert.True(t, exp.ExporterStarted)

	assert.Equal(t, 0, len(exp.Traces))
	err = exp.ConsumeTraces(context.Background(), pdata.Traces{})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(exp.Traces))

	assert.Equal(t, 0, len(exp.Metrics))
	err = exp.ConsumeMetrics(context.Background(), pdata.Metrics{})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(exp.Metrics))

	assert.False(t, exp.ExporterShutdown)
	err = exp.Shutdown(context.Background())
	assert.NoError(t, err)
	assert.True(t, exp.ExporterShutdown)
}
