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

	"go.opentelemetry.io/collector/consumer/pdata"
)

func TestTracesErr(t *testing.T) {
	err := errors.New("my error")
	nt := NewTracesErr(err)
	require.NotNil(t, nt)
	assert.Equal(t, err, nt.ConsumeTraces(context.Background(), pdata.NewTraces()))
}

func TestMetricsErr(t *testing.T) {
	err := errors.New("my error")
	nm := NewMetricsErr(err)
	require.NotNil(t, nm)
	assert.Equal(t, err, nm.ConsumeMetrics(context.Background(), pdata.NewMetrics()))
}

func TestLogsErr(t *testing.T) {
	err := errors.New("my error")
	nl := NewLogsErr(err)
	require.NotNil(t, nl)
	assert.Equal(t, err, nl.ConsumeLogs(context.Background(), pdata.NewLogs()))
}
