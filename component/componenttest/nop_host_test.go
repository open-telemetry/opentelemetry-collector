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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
)

func TestNewNopHost(t *testing.T) {
	nh := NewNopHost()
	require.NotNil(t, nh)
	require.IsType(t, &nopHost{}, nh)

	nh.ReportFatalError(errors.New("TestError"))
	assert.Nil(t, nh.GetExporters())
	assert.Nil(t, nh.GetExtensions())
	assert.Nil(t, nh.GetFactory(component.KindReceiver, "test"))
}
