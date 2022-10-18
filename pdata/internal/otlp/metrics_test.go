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

package otlp

import (
	"testing"

	"github.com/stretchr/testify/assert"

	otlpmetrics "go.opentelemetry.io/collector/pdata/internal/data/protogen/metrics/v1"
)

func TestDeprecatedScopeMetrics(t *testing.T) {
	sm := new(otlpmetrics.ScopeMetrics)
	rms := []*otlpmetrics.ResourceMetrics{
		{
			ScopeMetrics:           []*otlpmetrics.ScopeMetrics{sm},
			DeprecatedScopeMetrics: []*otlpmetrics.ScopeMetrics{sm},
		},
		{
			ScopeMetrics:           []*otlpmetrics.ScopeMetrics{},
			DeprecatedScopeMetrics: []*otlpmetrics.ScopeMetrics{sm},
		},
	}

	MigrateMetrics(rms)
	assert.Same(t, sm, rms[0].ScopeMetrics[0])
	assert.Same(t, sm, rms[1].ScopeMetrics[0])
	assert.Nil(t, rms[0].DeprecatedScopeMetrics)
	assert.Nil(t, rms[0].DeprecatedScopeMetrics)
}
