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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
)

func TestNewErrorWaitingHost(t *testing.T) {
	mh := NewErrorWaitingHost()
	require.NotNil(t, mh)

	reportedErr := errors.New("TestError")
	go mh.ReportFatalError(reportedErr)

	receivedError, receivedErr := mh.WaitForFatalError(100 * time.Millisecond)
	require.True(t, receivedError)
	require.Equal(t, reportedErr, receivedErr)

	receivedError, _ = mh.WaitForFatalError(100 * time.Millisecond)
	require.False(t, receivedError)
}

func TestNewErrorWaitingHost_Noop(t *testing.T) {
	mh := NewErrorWaitingHost()
	require.NotNil(t, mh)

	assert.Nil(t, mh.GetExporters())
	assert.Nil(t, mh.GetExtensions())
	assert.Nil(t, mh.GetFactory(component.KindReceiver, "test"))
}
