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

package exprfilterprocessor

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/processor/exprfilter"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

type processor struct {
	exclude *exprfilter.Matcher
}

var _ processorhelper.MProcessor = (*processor)(nil)

func newProcessor(cfg *Config) (*processor, error) {
	exclude, err := exprfilter.NewMatcher(cfg.Exclude[0])
	if err != nil {
		return nil, err
	}
	return &processor{exclude: exclude}, nil
}

func (l *processor) ProcessMetrics(_ context.Context, in pdata.Metrics) (pdata.Metrics, error) {
	exclusions := findLocations(in, l.exclude)
	if len(exclusions) > 0 {
		return filter(in, exclusions)
	}
	return in, nil
}

func findLocations(in pdata.Metrics, matcher *exprfilter.Matcher) locations {
	locs := locations{}
	rms := in.ResourceMetrics()
	for i := 0; i < rms.Len(); i++ {
		rm := rms.At(i)
		ilms := rm.InstrumentationLibraryMetrics()
		for j := 0; j < ilms.Len(); j++ {
			ilm := ilms.At(j)
			ms := ilm.Metrics()
			for k := 0; k < ms.Len(); k++ {
				metric := ms.At(k)
				if matcher.MatchMetric(metric) {
					locs.put(i, j, k)
				}
			}
		}
	}
	return locs
}

func filter(in pdata.Metrics, exclude locations) (pdata.Metrics, error) {
	out := pdata.NewMetrics()
	rmsOut := out.ResourceMetrics()
	rmsIn := in.ResourceMetrics()
	rmsOut.Resize(rmsIn.Len())
	for i := 0; i < rmsIn.Len(); i++ {
		rmIn := rmsIn.At(i)
		ilmsIn := rmIn.InstrumentationLibraryMetrics()
		rmOut := rmsOut.At(i)
		rmIn.Resource().CopyTo(rmOut.Resource())
		ilmsOut := rmOut.InstrumentationLibraryMetrics()
		ilmsOut.Resize(ilmsIn.Len())
		for j := 0; j < ilmsIn.Len(); j++ {
			ilmIn := ilmsIn.At(j)
			msIn := ilmIn.Metrics()
			ilmOut := ilmsOut.At(j)
			msOut := ilmOut.Metrics()
			for k := 0; k < msIn.Len(); k++ {
				metricIn := msIn.At(k)
				if !exclude.contains(i, j, k) {
					msOut.Append(metricIn)
				}
			}
		}
	}
	return out, nil
}

type locations map[string]bool

func (l locations) put(i, j, k int) {
	l[key(i, j, k)] = true
}

func (l locations) contains(i, j, k int) bool {
	return l[key(i, j, k)]
}

func key(i, j, k int) string {
	return fmt.Sprintf("%d-%d-%d", i, j, k)
}
