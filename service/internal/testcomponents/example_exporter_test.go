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
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func TestExampleExporter(t *testing.T) {
	exp := &ExampleExporter{}
	host := componenttest.NewNopHost()
	assert.False(t, exp.Started())
	assert.NoError(t, exp.Start(context.Background(), host))
	assert.True(t, exp.Started())

	assert.Equal(t, 0, len(exp.traces))
	assert.NoError(t, exp.ConsumeTraces(context.Background(), ptrace.Traces{}))
	assert.Equal(t, 1, len(exp.traces))

	assert.Equal(t, 0, len(exp.metrics))
	assert.NoError(t, exp.ConsumeMetrics(context.Background(), pmetric.Metrics{}))
	assert.Equal(t, 1, len(exp.metrics))

	assert.Equal(t, 0, len(exp.logs))
	assert.NoError(t, exp.ConsumeLogs(context.Background(), plog.Logs{}))
	assert.Equal(t, 1, len(exp.logs))

	assert.False(t, exp.Stopped())
	assert.NoError(t, exp.Shutdown(context.Background()))
	assert.True(t, exp.Stopped())
}
