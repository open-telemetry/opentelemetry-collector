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

package consumertest

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func TestErr(t *testing.T) {
	err := errors.New("my error")
	ec := NewErr(err)
	require.NotNil(t, ec)
	assert.NotPanics(t, ec.unexported)
	assert.Equal(t, err, ec.ConsumeLogs(context.Background(), plog.NewLogs()))
	assert.Equal(t, err, ec.ConsumeMetrics(context.Background(), pmetric.NewMetrics()))
	assert.Equal(t, err, ec.ConsumeTraces(context.Background(), ptrace.NewTraces()))
}
