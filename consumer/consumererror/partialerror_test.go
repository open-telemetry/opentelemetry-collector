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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/internal/testdata"
)

func TestPartialError(t *testing.T) {
	td := testdata.GenerateTraceDataOneSpan()
	err := fmt.Errorf("some error")
	partialErr := PartialTracesError(err, td)
	assert.Equal(t, err.Error(), partialErr.Error())
	assert.Equal(t, td, partialErr.(PartialError).failed)
}

func TestPartialErrorLogs(t *testing.T) {
	td := testdata.GenerateLogDataOneLog()
	err := fmt.Errorf("some error")
	partialErr := PartialLogsError(err, td)
	assert.Equal(t, err.Error(), partialErr.Error())
	assert.Equal(t, td, partialErr.(PartialError).failedLogs)
}

func TestPartialErrorMetrics(t *testing.T) {
	td := testdata.GenerateMetricsOneMetric()
	err := fmt.Errorf("some error")
	partialErr := PartialMetricsError(err, td)
	assert.Equal(t, err.Error(), partialErr.Error())
	assert.Equal(t, td, partialErr.(PartialError).failedMetrics)
}
