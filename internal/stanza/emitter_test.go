// Copyright The OpenTelemetry Authors
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

package stanza

import (
	"context"
	"testing"
	"time"

	"github.com/open-telemetry/opentelemetry-log-collection/entry"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestLogEmitter(t *testing.T) {

	emitter := NewLogEmitter(zaptest.NewLogger(t).Sugar())
	defer emitter.Stop()

	in := entry.New()

	go func() {
		require.NoError(t, emitter.Process(context.Background(), in))
	}()

	select {
	case out := <-emitter.logChan:
		require.Equal(t, in, out)
	case <-time.After(time.Second):
		require.FailNow(t, "Timed out waiting for output")
	}
}
