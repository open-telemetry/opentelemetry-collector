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

package filterexpr

import (
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
	HasLabel   func(key string) bool
	Label      func(key string) string
}

func NewMatcher(expression string) (*Matcher, error) {
	program, err := expr.Compile(expression)
	if err != nil {
		return nil, err
	}
	return &Matcher{program: program, v: vm.VM{}}, nil
}

func (m *Matcher) MatchMetric(metric pdata.Metric) (bool, error) {
	metricName := metric.Name()
	switch metric.DataType() {
	case pdata.MetricDataTypeIntGauge:
		return m.matchIntGauge(metricName, metric.IntGauge())
	case pdata.MetricDataTypeDoubleGauge:
		return m.matchDoubleGauge(metricName, metric.DoubleGauge())
	case pdata.MetricDataTypeIntSum:
		return m.matchIntSum(metricName, metric.IntSum())
	case pdata.MetricDataTypeDoubleSum:
		return m.matchDoubleSum(metricName, metric.DoubleSum())
	case pdata.MetricDataTypeIntHistogram:
		return m.matchIntHistogram(metricName, metric.IntHistogram())
	case pdata.MetricDataTypeDoubleHistogram:
		return m.matchDoubleHistogram(metricName, metric.DoubleHistogram())
	default:
		return false, nil
	}
}

func (m *Matcher) matchIntGauge(metricName string, gauge pdata.IntGauge) (bool, error) {
	if gauge.IsNil() {
		return false, nil
	}
	pts := gauge.DataPoints()
	for i := 0; i < pts.Len(); i++ {
		pt := pts.At(i)
		if pt.IsNil() {
			continue
		}
		matched, err := m.matchEnv(metricName, pt.LabelsMap())
		if err != nil {
			return false, err
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}

func (m *Matcher) matchDoubleGauge(metricName string, gauge pdata.DoubleGauge) (bool, error) {
	if gauge.IsNil() {
		return false, nil
	}
	pts := gauge.DataPoints()
	for i := 0; i < pts.Len(); i++ {
		pt := pts.At(i)
		if pt.IsNil() {
			continue
		}
		matched, err := m.matchEnv(metricName, pt.LabelsMap())
		if err != nil {
			return false, err
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}

func (m *Matcher) matchDoubleSum(metricName string, sum pdata.DoubleSum) (bool, error) {
	if sum.IsNil() {
		return false, nil
	}
	pts := sum.DataPoints()
	for i := 0; i < pts.Len(); i++ {
		pt := pts.At(i)
		if pt.IsNil() {
			continue
		}
		matched, err := m.matchEnv(metricName, pt.LabelsMap())
		if err != nil {
			return false, err
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}

func (m *Matcher) matchIntSum(metricName string, sum pdata.IntSum) (bool, error) {
	if sum.IsNil() {
		return false, nil
	}
	pts := sum.DataPoints()
	for i := 0; i < pts.Len(); i++ {
		pt := pts.At(i)
		if pt.IsNil() {
			continue
		}
		matched, err := m.matchEnv(metricName, pt.LabelsMap())
		if err != nil {
			return false, err
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}

func (m *Matcher) matchIntHistogram(metricName string, histogram pdata.IntHistogram) (bool, error) {
	if histogram.IsNil() {
		return false, nil
	}
	pts := histogram.DataPoints()
	for i := 0; i < pts.Len(); i++ {
		pt := pts.At(i)
		if pt.IsNil() {
			continue
		}
		matched, err := m.matchEnv(metricName, pt.LabelsMap())
		if err != nil {
			return false, err
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}

func (m *Matcher) matchDoubleHistogram(metricName string, histogram pdata.DoubleHistogram) (bool, error) {
	if histogram.IsNil() {
		return false, nil
	}
	pts := histogram.DataPoints()
	for i := 0; i < pts.Len(); i++ {
		pt := pts.At(i)
		if pt.IsNil() {
			continue
		}
		matched, err := m.matchEnv(metricName, pt.LabelsMap())
		if err != nil {
			return false, err
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}

func (m *Matcher) matchEnv(metricName string, labelsMap pdata.StringMap) (bool, error) {
	return m.match(createEnv(metricName, labelsMap))
}

func createEnv(metricName string, labelsMap pdata.StringMap) env {
	return env{
		MetricName: metricName,
		HasLabel: func(key string) bool {
			_, ok := labelsMap.Get(key)
			return ok
		},
		Label: func(key string) string {
			v, ok := labelsMap.Get(key)
			if !ok {
				return ""
			}
			return v.Value()
		},
	}
}

func (m *Matcher) match(env env) (bool, error) {
	result, err := m.v.Run(m.program, env)
	if err != nil {
		return false, err
	}
	return result.(bool), nil
}
