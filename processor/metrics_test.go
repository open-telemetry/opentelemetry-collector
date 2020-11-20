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

package processor

import (
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/internal/testdata"
)

func TestSpanCountByResourceStringAttribute(t *testing.T) {
	td := testdata.GenerateTraceDataEmpty()
	require.EqualValues(t, 0, len(spanCountByResourceStringAttribute(td, "resource-attr")))

	td = testdata.GenerateTraceDataOneSpan()
	spanCounts := spanCountByResourceStringAttribute(td, "resource-attr")
	require.EqualValues(t, 1, len(spanCounts))
	require.EqualValues(t, 1, spanCounts["resource-attr-val-1"])

	td = testdata.GenerateTraceDataTwoSpansSameResource()
	spanCounts = spanCountByResourceStringAttribute(td, "resource-attr")
	require.EqualValues(t, 1, len(spanCounts))
	require.EqualValues(t, 2, spanCounts["resource-attr-val-1"])

	td = testdata.GenerateTraceDataTwoSpansSameResourceOneDifferent()
	spanCounts = spanCountByResourceStringAttribute(td, "resource-attr")
	require.EqualValues(t, 2, len(spanCounts))
	require.EqualValues(t, 2, spanCounts["resource-attr-val-1"])
	require.EqualValues(t, 1, spanCounts["resource-attr-val-2"])
}
