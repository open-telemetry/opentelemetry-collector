// Copyright 2019, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package stanzareceiver

import (
	"context"
	"testing"
	"time"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestChannelOperator(t *testing.T) {
	cfg := defaultCfg()

	logsChan := make(chan *entry.Entry)
	defer close(logsChan)

	buildContext := testutil.NewBuildContext(t)
	buildContext.Parameters = map[string]interface{}{"logs_channel": logsChan}
	operator, err := cfg.Build(buildContext)
	require.NoError(t, err)

	in := entry.New()

	go func() {
		err = operator.Process(context.Background(), in)
		require.NoError(t, err)
	}()

	select {
	case out := <-logsChan:
		require.Equal(t, in, out)
	case <-time.After(time.Second):
		require.FailNow(t, "Timed out waiting for output")
	}
}

func TestChannelOperatorBuildFailures(t *testing.T) {
	cfg := defaultCfg()

	buildContext := operator.BuildContext{}

	_, err := cfg.Build(buildContext)
	require.Error(t, err, "Build requires a logger")

	buildContext.Logger = zaptest.NewLogger(t).Sugar()
	_, err = cfg.Build(buildContext)
	require.Error(t, err, "Build requires a parameters map")

	intChan := make(chan int)
	defer close(intChan)

	buildContext.Parameters = map[string]interface{}{"int_channel": intChan}
	_, err = cfg.Build(buildContext)
	require.Error(t, err, "Build requires a 'logs_channel' parameter")

	buildContext.Parameters = map[string]interface{}{"logs_channel": intChan}
	_, err = cfg.Build(buildContext)
	require.Error(t, err, "Build requires a 'logs_channel' parameter to be a logs channel")
}
