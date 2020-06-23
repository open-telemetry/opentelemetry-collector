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

package goldendataset

import (
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/pdata"
)

func TestGenerateGoldenMetricsBasic(t *testing.T) {
	md, err := GenerateMetrics(&PICTMetricInputs{
		MetricInputs: AttrsOne,
	})
	require.NoError(t, err)
	require.NotNil(t, md)

	rmSlice := md.ResourceMetrics()
	require.Equal(t, 1, rmSlice.Len())

	rm := rmSlice.At(0)
	require.NotNil(t, rm)

	rsrc := rm.Resource()
	require.NotNil(t, rsrc)
	require.False(t, rsrc.IsNil())

	attrs := rsrc.Attributes()
	rsrcName, ok := attrs.Get("my-name")
	require.True(t, ok)
	require.Equal(t, "resourceName", rsrcName.StringVal())

	ilmSlice := rm.InstrumentationLibraryMetrics()
	require.NotNil(t, ilmSlice)
	require.Equal(t, 1, ilmSlice.Len())

	ilm := ilmSlice.At(0)
	require.NotNil(t, ilm)
	require.False(t, ilm.IsNil())

	metricSlice := ilm.Metrics()
	require.Equal(t, 1, metricSlice.Len())

	metric := metricSlice.At(0)
	require.NotNil(t, metric)

	mDesc := metric.MetricDescriptor()

	require.NotNil(t, mDesc)
	mdName := mDesc.Name()
	require.Equal(t, "my-md-name", mdName)

	desc := mDesc.Description()
	require.Equal(t, "my-md-desc", desc)

	mdt := mDesc.Type()
	require.Equal(t, pdata.MetricTypeInt64, mdt)

	ptSlice := metric.Int64DataPoints()
	pt := ptSlice.At(0)

	st := pt.StartTime()
	require.Equal(t, pdata.TimestampUnixNano(uint64(128)), st)

	ptVal := pt.Value()
	require.Equal(t, int64(42), ptVal)

	lm := pt.LabelsMap()
	lmVal, ok := lm.Get("lblKey")
	require.True(t, ok)
	require.Equal(t, "lblVal", lmVal.Value())
}

func TestGenerateGoldenMetricsNoAttrs(t *testing.T) {
	md, err := GenerateMetrics(&PICTMetricInputs{
		MetricInputs: AttrsNone,
	})
	require.NoError(t, err)
	attrs := md.ResourceMetrics().At(0).Resource().Attributes()
	require.Equal(t, 0, attrs.Len())
}

func TestGenerateGoldenMetricsOneAttr(t *testing.T) {
	md, err := GenerateMetrics(&PICTMetricInputs{
		MetricInputs: AttrsOne,
	})
	require.NoError(t, err)
	attrs := md.ResourceMetrics().At(0).Resource().Attributes()
	require.Equal(t, 1, attrs.Len())
}

func TestGenerateGoldenMetricsTwoAttrs(t *testing.T) {
	md, err := GenerateMetrics(&PICTMetricInputs{
		MetricInputs: AttrsTwo,
	})
	require.NoError(t, err)
	attrs := md.ResourceMetrics().At(0).Resource().Attributes()
	// key is repeated so we only get one
	require.Equal(t, 1, attrs.Len())
}
