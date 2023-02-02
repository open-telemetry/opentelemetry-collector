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

package ptrace // import "go.opentelemetry.io/collector/pdata/ptrace"

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSpanKindString(t *testing.T) {
	assert.EqualValues(t, "Unspecified", SpanKindUnspecified.String())
	assert.EqualValues(t, "Internal", SpanKindInternal.String())
	assert.EqualValues(t, "Server", SpanKindServer.String())
	assert.EqualValues(t, "Client", SpanKindClient.String())
	assert.EqualValues(t, "Producer", SpanKindProducer.String())
	assert.EqualValues(t, "Consumer", SpanKindConsumer.String())
	assert.EqualValues(t, "", SpanKind(100).String())
}
