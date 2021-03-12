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
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/component"
)

// assertNoErrorHost implements a component.Host that asserts that
// there were no errors.
type assertNoErrorHost struct {
	nopHost
	t *testing.T
}

var _ component.Host = (*assertNoErrorHost)(nil)

// NewAssertNoError returns a new instance of assertNoErrorHost.
func NewAssertNoError(t *testing.T) component.Host {
	return &assertNoErrorHost{
		t: t,
	}
}

func (aneh *assertNoErrorHost) ReportFatalError(err error) {
	assert.NoError(aneh.t, err)
}
