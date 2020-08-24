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
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/scrape"
	"go.uber.org/zap"
)

// test helpers

var testLogger *zap.Logger

func init() {
	zl, _ := zap.NewDevelopment()
	testLogger = zl
}

type mockMetadataCache struct {
	data map[string]scrape.MetricMetadata
}

func newMockMetadataCache(data map[string]scrape.MetricMetadata) *mockMetadataCache {
	return &mockMetadataCache{data: data}
}

func (m *mockMetadataCache) Metadata(metricName string) (scrape.MetricMetadata, bool) {
	mm, ok := m.data[metricName]
	return mm, ok
}

func (m *mockMetadataCache) SharedLabels() labels.Labels {
	return labels.FromStrings("__scheme__", "http")
}

type mockScrapeManager struct {
	targets map[string][]*scrape.Target
}

func (sm *mockScrapeManager) TargetsAll() map[string][]*scrape.Target {
	return sm.targets
}
