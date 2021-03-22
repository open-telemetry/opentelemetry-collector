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

package consumererror

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/internal/testdata"
)

func TestPartialErrorTraces(t *testing.T) {
	td := testdata.GenerateTraceDataOneSpan()
	err := fmt.Errorf("some error")
	partialErr := PartialTraces(err, td)
	assert.True(t, IsPartial(partialErr))
	assert.Equal(t, err.Error(), partialErr.Error())
	assert.Equal(t, td, partialErr.(PartialError).GetTraces())
}

func TestPartialErrorLogs(t *testing.T) {
	td := testdata.GenerateLogDataOneLog()
	err := fmt.Errorf("some error")
	partialErr := PartialLogs(err, td)
	assert.True(t, IsPartial(partialErr))
	assert.Equal(t, err.Error(), partialErr.Error())
	assert.Equal(t, td, partialErr.(PartialError).GetLogs())
}

func TestPartialErrorMetrics(t *testing.T) {
	td := testdata.GenerateMetricsOneMetric()
	err := fmt.Errorf("some error")
	partialErr := PartialMetrics(err, td)
	assert.True(t, IsPartial(partialErr))
	assert.Equal(t, err.Error(), partialErr.Error())
	assert.Equal(t, td, partialErr.(PartialError).GetMetrics())
}

func TestIsPartialNilInput(t *testing.T) {
	assert.False(t, IsPartial(nil))
}

func BenchmarkIsPartialNil(b *testing.B) {
	for n := 0; n < b.N; n++ {
		IsPartial(nil)
	}
}

func BenchmarkIsPartialFalse(b *testing.B) {
	err := errors.New("Test error")
	for n := 0; n < b.N; n++ {
		IsPartial(err)
	}
}

func BenchmarkIsPartialTrue(b *testing.B) {
	td := testdata.GenerateTraceDataOneSpan()
	err := fmt.Errorf("some error")
	partialErr := PartialTraces(err, td)
	for n := 0; n < b.N; n++ {
		IsPartial(partialErr)
	}
}
