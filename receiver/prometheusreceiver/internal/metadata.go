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

package internal

import (
	"errors"

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/scrape"
)

// MetadataService is an adapter to scrapeManager and provide only the functionality which is needed
type MetadataService interface {
	Get(job, instance string) (MetadataCache, error)
}

// MetadataCache is an adapter to prometheus' scrape.Target  and provide only the functionality which is needed
type MetadataCache interface {
	Metadata(metricName string) (scrape.MetricMetadata, bool)
	SharedLabels() labels.Labels
}

type mService struct {
	sm *scrape.Manager
}

func (t *mService) Get(job, instance string) (MetadataCache, error) {
	targetGroup, ok := t.sm.TargetsAll()[job]
	if !ok {
		return nil, errors.New("unable to find a target group with job=" + job)
	}

	// from the same targetGroup, instance is not going to be duplicated
	for _, target := range targetGroup {
		if target.DiscoveredLabels().Get(model.AddressLabel) == instance {
			return &mCache{target}, nil
		}
	}

	return nil, errors.New("unable to find a target with job=" + job + ", and instance=" + instance)
}

// adapter to get metadata from scrape.Target
type mCache struct {
	t *scrape.Target
}

func (m *mCache) Metadata(metricName string) (scrape.MetricMetadata, bool) {
	return m.t.Metadata(metricName)
}

func (m *mCache) SharedLabels() labels.Labels {
	return m.t.DiscoveredLabels()
}
