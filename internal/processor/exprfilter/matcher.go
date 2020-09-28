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

package exprfilter

import (
	"log"

	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/vm"

	"go.opentelemetry.io/collector/consumer/pdata"
)

type Matcher struct {
	program *vm.Program
	v       vm.VM
}

type env struct {
	MetricName string
	LabelsMap  map[string]string
}

func NewMatcher(searchStr string) (*Matcher, error) {
	program, err := expr.Compile(searchStr)
	if err != nil {
		return nil, err
	}
	return &Matcher{program: program, v: vm.VM{}}, nil
}

func (m *Matcher) MatchMetric(metric pdata.Metric) bool {
	metricName := metric.Name()
	switch metric.DataType() {
	case pdata.MetricDataTypeIntGauge:
		pts := metric.IntGauge().DataPoints()
		for i := 0; i < pts.Len(); i++ {
			pt := pts.At(i)
			lbls := map[string]string{}
			pt.LabelsMap().ForEach(func(k string, v pdata.StringValue) {
				lbls[k] = v.Value()
			})
			if m.match(env{MetricName: metricName, LabelsMap: lbls}) {
				return true
			}
		}
	case pdata.MetricDataTypeDoubleGauge:
	case pdata.MetricDataTypeIntSum:
	case pdata.MetricDataTypeDoubleSum:
	case pdata.MetricDataTypeIntHistogram:
	case pdata.MetricDataTypeDoubleHistogram:
	}
	return false
}

func (m *Matcher) match(env env) bool {
	result, err := m.v.Run(m.program, env)
	if err != nil {
		log.Printf("expr run error: %s", err.Error())
		return false
	}
	return result.(bool)
}
