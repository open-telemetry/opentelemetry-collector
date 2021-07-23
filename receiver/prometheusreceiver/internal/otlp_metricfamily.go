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
	"sort"
	"strings"

	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/pkg/textparse"

	"go.opentelemetry.io/collector/model/pdata"
)

type metricFamilyPdata struct {
	// We are composing the already present metricFamily to
	// make for a scalable migration, so that we only edit target
	// fields progressively, when we are ready to make changes.
	metricFamily
	mtype  pdata.MetricDataType
	groups map[string]*metricGroupPdata
}

// metricGroupPdata, represents a single metric of a metric family. for example a histogram metric is usually represent by
// a couple data complexValue (buckets and count/sum), a group of a metric family always share a same set of tags. for
// simple types like counter and gauge, each data point is a group of itself
type metricGroupPdata struct {
	// We are composing the already present metricGroup to
	// make for a scalable migration, so that we only edit target
	// fields progressively, when we are ready to make changes.
	metricGroup
	family *metricFamilyPdata
}

func newMetricFamilyPdata(metricName string, mc MetadataCache) MetricFamily {
	familyName := normalizeMetricName(metricName)

	// lookup metadata based on familyName
	metadata, ok := mc.Metadata(familyName)
	if !ok && metricName != familyName {
		// use the original metricName as metricFamily
		familyName = metricName
		// perform a 2nd lookup with the original metric name. it can happen if there's a metric which is not histogram
		// or summary, but ends with one of those _count/_sum suffixes
		metadata, ok = mc.Metadata(metricName)
		// still not found, this can happen when metric has no TYPE HINT
		if !ok {
			metadata.Metric = familyName
			metadata.Type = textparse.MetricTypeUnknown
		}
	}

	return &metricFamilyPdata{
		mtype:  convToPdataMetricType(metadata.Type),
		groups: make(map[string]*metricGroupPdata),
		metricFamily: metricFamily{
			name:              familyName,
			mc:                mc,
			droppedTimeseries: 0,
			labelKeys:         make(map[string]bool),
			labelKeysOrdered:  make([]string, 0),
			metadata:          &metadata,
			groupOrders:       make(map[string]int),
		},
	}
}

// updateLabelKeys is used to store all the label keys of a same metric family in observed order. since prometheus
// receiver removes any label with empty value before feeding it to an appender, in order to figure out all the labels
// from the same metric family we will need to keep track of what labels have ever been observed.
func (mf *metricFamilyPdata) updateLabelKeys(ls labels.Labels) {
	for _, l := range ls {
		if isUsefulLabelPdata(mf.mtype, l.Name) {
			if _, ok := mf.labelKeys[l.Name]; !ok {
				mf.labelKeys[l.Name] = true
				// use insertion sort to maintain order
				i := sort.SearchStrings(mf.labelKeysOrdered, l.Name)
				mf.labelKeysOrdered = append(mf.labelKeysOrdered, "")
				copy(mf.labelKeysOrdered[i+1:], mf.labelKeysOrdered[i:])
				mf.labelKeysOrdered[i] = l.Name

			}
		}
	}
}

// Purposefully being referenced to avoid lint warnings about being "unused".
var _ = (*metricFamilyPdata)(nil).updateLabelKeys

func (mf *metricFamilyPdata) isCumulativeTypePdata() bool {
	return mf.mtype == pdata.MetricDataTypeSum ||
		mf.mtype == pdata.MetricDataTypeIntSum ||
		mf.mtype == pdata.MetricDataTypeHistogram ||
		mf.mtype == pdata.MetricDataTypeSummary
}

func (mf *metricFamilyPdata) loadMetricGroupOrCreate(groupKey string, ls labels.Labels, ts int64) *metricGroupPdata {
	mg, ok := mf.groups[groupKey]
	if !ok {
		mg = &metricGroupPdata{
			family: mf,
			metricGroup: metricGroup{
				ls:           ls,
				ts:           ts,
				complexValue: make([]*dataPoint, 0),
			},
		}
		mf.groups[groupKey] = mg
		// maintaining data insertion order is helpful to generate stable/reproducible metric output
		mf.groupOrders[groupKey] = len(mf.groupOrders)
	}
	return mg
}

func (mf *metricFamilyPdata) Add(metricName string, ls labels.Labels, t int64, v float64) error {
	groupKey := mf.getGroupKey(ls)
	mg := mf.loadMetricGroupOrCreate(groupKey, ls, t)
	switch mf.mtype {
	case pdata.MetricDataTypeHistogram, pdata.MetricDataTypeSummary:
		switch {
		case strings.HasSuffix(metricName, metricsSuffixSum):
			// always use the timestamp from sum (count is ok too), because the startTs from quantiles won't be reliable
			// in cases like remote server restart
			mg.ts = t
			mg.sum = v
			mg.hasSum = true
		case strings.HasSuffix(metricName, metricsSuffixCount):
			mg.count = v
			mg.hasCount = true
		default:
			boundary, err := getBoundaryPdata(mf.mtype, ls)
			if err != nil {
				mf.droppedTimeseries++
				return err
			}
			mg.complexValue = append(mg.complexValue, &dataPoint{value: v, boundary: boundary})
		}
	default:
		mg.value = v
	}

	return nil
}
