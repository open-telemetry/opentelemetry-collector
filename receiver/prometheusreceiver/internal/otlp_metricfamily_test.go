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

package internal

import (
	"testing"

	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/pkg/textparse"
	"github.com/prometheus/prometheus/scrape"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type byLookupMetadataCache map[string]scrape.MetricMetadata

func (bmc byLookupMetadataCache) Metadata(familyName string) (scrape.MetricMetadata, bool) {
	lookup, ok := bmc[familyName]
	return lookup, ok
}

func (bmc byLookupMetadataCache) SharedLabels() labels.Labels {
	return nil
}

func TestIsCumulativeEquivalence(t *testing.T) {
	mc := byLookupMetadataCache{
		"counter": scrape.MetricMetadata{
			Metric: "cr",
			Type:   textparse.MetricTypeCounter,
			Help:   "This is some help",
			Unit:   "By",
		},
		"gauge": scrape.MetricMetadata{
			Metric: "ge",
			Type:   textparse.MetricTypeGauge,
			Help:   "This is some help",
			Unit:   "1",
		},
		"gaugehistogram": scrape.MetricMetadata{
			Metric: "gh",
			Type:   textparse.MetricTypeGaugeHistogram,
			Help:   "This is some help",
			Unit:   "?",
		},
		"histogram": scrape.MetricMetadata{
			Metric: "hg",
			Type:   textparse.MetricTypeHistogram,
			Help:   "This is some help",
			Unit:   "ms",
		},
		"summary": scrape.MetricMetadata{
			Metric: "s",
			Type:   textparse.MetricTypeSummary,
			Help:   "This is some help",
			Unit:   "?",
		},
		"unknown": scrape.MetricMetadata{
			Metric: "u",
			Type:   textparse.MetricTypeUnknown,
			Help:   "This is some help",
			Unit:   "?",
		},
	}

	tests := []struct {
		name string
		want bool
	}{
		{name: "counter", want: true},
		{name: "gauge", want: false},
		{name: "histogram", want: true},
		{name: "gaugehistogram", want: false},
		{name: "does not exist", want: false},
		{name: "unknown", want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mf := newMetricFamily(tt.name, mc, zap.NewNop()).(*metricFamily)
			mfp := newMetricFamilyPdata(tt.name, mc).(*metricFamilyPdata)
			assert.Equal(t, mf.isCumulativeType(), mfp.isCumulativeTypePdata(), "mismatch in isCumulative")
			assert.Equal(t, mf.isCumulativeType(), tt.want, "isCumulative does not match for regular metricFamily")
			assert.Equal(t, mfp.isCumulativeTypePdata(), tt.want, "isCumulative does not match for pdata metricFamily")
		})
	}
}
