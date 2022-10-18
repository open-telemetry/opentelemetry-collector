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

	otlplogs "go.opentelemetry.io/collector/pdata/internal/data/protogen/logs/v1"
)

func TestDeprecatedScopeLogs(t *testing.T) {
	sl := new(otlplogs.ScopeLogs)
	rls := []*otlplogs.ResourceLogs{
		{
			ScopeLogs:           []*otlplogs.ScopeLogs{sl},
			DeprecatedScopeLogs: []*otlplogs.ScopeLogs{sl},
		},
		{
			ScopeLogs:           []*otlplogs.ScopeLogs{},
			DeprecatedScopeLogs: []*otlplogs.ScopeLogs{sl},
		},
	}

	MigrateLogs(rls)
	assert.Same(t, sl, rls[0].ScopeLogs[0])
	assert.Same(t, sl, rls[1].ScopeLogs[0])
	assert.Nil(t, rls[0].DeprecatedScopeLogs)
	assert.Nil(t, rls[0].DeprecatedScopeLogs)
}
