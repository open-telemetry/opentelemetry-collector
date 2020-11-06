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
package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

func createTagKey(name string) tag.Key {
	key, err := tag.NewKey(name)
	if err != nil {
		panic("can't create a tag")
	}
	return key
}

func createViewData() *view.Data {
	vd := &view.Data{
		View: &view.View{
			Name:        "metric_name",
			Measure:     stats.Int64("metric_name", "", "By"),
			Description: "",
		},
		Rows: []*view.Row{
			{
				Data: &view.SumData{Value: 123},
			},
		},
	}
	return vd
}

func createViewData2Tags() *view.Data {
	vd := &view.Data{
		View: &view.View{
			Name:        "process_time",
			Measure:     stats.Float64("process_time", "", ""),
			Description: "",
		},
		Rows: []*view.Row{
			{
				Tags: []tag.Tag{
					{Key: createTagKey("exporter"), Value: "otlp"},
					{Key: createTagKey("component"), Value: "exporter"},
				},
				Data: &view.LastValueData{Value: 123.45},
			},
		},
	}
	return vd
}

func Test_viewDataToRecord(t *testing.T) {
	lr := viewDataToRecord(createViewData())

	assert.EqualValues(t, "", lr.tags)
	assert.EqualValues(t, "metric_name", lr.name)
	assert.EqualValues(t, "         123 By", lr.value)

	lr = viewDataToRecord(createViewData2Tags())

	assert.EqualValues(t, "exporter=otlp, component=exporter", lr.tags)
	assert.EqualValues(t, "process_time", lr.name)
	assert.EqualValues(t, "  123.450000", lr.value)
}

func Test_recordsToStr(t *testing.T) {
	lr1 := viewDataToRecord(createViewData())
	lr2 := viewDataToRecord(createViewData2Tags())

	str := recordsToStr([]metricLogRecord{lr1, lr2})
	assert.EqualValues(t, `
  Internal Metrics:
  Metric                                            | Value
  --------------------------------------------------|--------------------------------
  metric_name                                       |          123 By
  --------------------------------------------------|--------------------------------

  Component/Dimensions                              | Metric                                  | Value
  --------------------------------------------------|-----------------------------------------|--------------------------------
  exporter=otlp, component=exporter                 | process_time                            |   123.450000
  --------------------------------------------------|-----------------------------------------|--------------------------------`,
		str)
}
