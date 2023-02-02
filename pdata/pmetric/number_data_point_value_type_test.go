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

package pmetric // import "go.opentelemetry.io/collector/pdata/pmetric"

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNumberDataPointValueTypeString(t *testing.T) {
	assert.Equal(t, "Empty", NumberDataPointValueTypeEmpty.String())
	assert.Equal(t, "Int", NumberDataPointValueTypeInt.String())
	assert.Equal(t, "Double", NumberDataPointValueTypeDouble.String())
	assert.Equal(t, "", (NumberDataPointValueTypeDouble + 1).String())
}
