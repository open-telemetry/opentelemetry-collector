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

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/processor/exprfilter"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

type processor struct {
	matcher *exprfilter.Matcher
}

var _ processorhelper.MProcessor = (*processor)(nil)

func newProcessor(cfg *Config) (*processor, error) {
	exclude, err := exprfilter.NewMatcher(cfg.Query)
	if err != nil {
		return nil, err
	}
	return &processor{matcher: exclude}, nil
}

func (p *processor) ProcessMetrics(_ context.Context, in pdata.Metrics) (pdata.Metrics, error) {
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
				if p.matcher.MatchMetric(metricIn) {
					msOut.Append(metricIn)
				}
			}
		}
	}
	return out, nil
}
