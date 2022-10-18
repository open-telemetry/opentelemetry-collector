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

	otlptrace "go.opentelemetry.io/collector/pdata/internal/data/protogen/trace/v1"
)

func TestDeprecatedScopeSpans(t *testing.T) {
	ss := new(otlptrace.ScopeSpans)
	rss := []*otlptrace.ResourceSpans{
		{
			ScopeSpans:           []*otlptrace.ScopeSpans{ss},
			DeprecatedScopeSpans: []*otlptrace.ScopeSpans{ss},
		},
		{
			ScopeSpans:           []*otlptrace.ScopeSpans{},
			DeprecatedScopeSpans: []*otlptrace.ScopeSpans{ss},
		},
	}

	MigrateTraces(rss)
	assert.Same(t, ss, rss[0].ScopeSpans[0])
	assert.Same(t, ss, rss[1].ScopeSpans[0])
	assert.Nil(t, rss[0].DeprecatedScopeSpans)
	assert.Nil(t, rss[0].DeprecatedScopeSpans)
}
