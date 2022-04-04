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

package component

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/config"
)

func TestStatus_StatusReporters(t *testing.T) {
	reporters := NewStatusReporters()

	expectedStatus := StatusReport{
		ComponentID: config.NewComponentID("nop"),
		Error:       errors.New("an error"),
	}

	var f1Called, f2Called bool

	f1 := func(status StatusReport) {
		require.Equal(t, expectedStatus, status)
		f1Called = true
	}

	f2 := func(status StatusReport) {
		require.Equal(t, expectedStatus, status)
		f2Called = true
	}

	reporters.Start()
	reporters.Register(f1)
	reporters.Register(f2)
	reporters.Report(expectedStatus)

	require.Eventually(t, func() bool {
		return f1Called && f2Called
	}, time.Second, time.Microsecond)

	reporters.Stop()
}
