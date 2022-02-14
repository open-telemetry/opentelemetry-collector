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

package zpages

import (
	"bytes"
	"html/template"
	"testing"

	"github.com/stretchr/testify/assert"
)

const tmplBody = `
        <p>{{.Index|even}}</p>
        <p>{{.Element|getKey}}</p>
        <p>{{.Element|getValue}}</p>
`

const want = `
        <p>true</p>
        <p>key</p>
        <p>value</p>
`

type testFuncsInput struct {
	Index   int
	Element [2]string
}

var tmpl = template.Must(template.New("countTest").Funcs(templateFunctions).Parse(tmplBody))

func TestTemplateFuncs(t *testing.T) {
	buf := new(bytes.Buffer)
	input := testFuncsInput{
		Index:   32,
		Element: [2]string{"key", "value"},
	}
	assert.NoError(t, tmpl.Execute(buf, input))
	assert.EqualValues(t, want, buf.String())
}

func TestNoCrash(t *testing.T) {
	buf := new(bytes.Buffer)
	assert.NotPanics(t, func() { WriteHTMLPageHeader(buf, HeaderData{Title: "Foo"}) })
	assert.NotPanics(t, func() { WriteHTMLComponentHeader(buf, ComponentHeaderData{Name: "Bar"}) })
	assert.NotPanics(t, func() {
		WriteHTMLComponentHeader(buf, ComponentHeaderData{Name: "Bar", ComponentEndpoint: "pagez", Link: true})
	})
	assert.NotPanics(t, func() {
		WriteHTMLPipelinesSummaryTable(buf, SummaryPipelinesTableData{
			Rows: []SummaryPipelinesTableRowData{{
				FullName:    "test",
				InputType:   "metrics",
				MutatesData: false,
				Receivers:   []string{"oc"},
				Processors:  []string{"nop"},
				Exporters:   []string{"oc"},
			}},
		})
	})
	assert.NotPanics(t, func() {
		WriteHTMLExtensionsSummaryTable(buf, SummaryExtensionsTableData{
			Rows: []SummaryExtensionsTableRowData{{
				FullName: "test",
			}},
		})
	})
	assert.NotPanics(t, func() {
		WriteHTMLPropertiesTable(buf, PropertiesTableData{Name: "Bar", Properties: [][2]string{{"key", "value"}}})
	})
	assert.NotPanics(t, func() {
		WriteHTMLFeaturesTable(buf, FeatureGateTableData{Rows: []FeatureGateTableRowData{
			{
				ID:          "test",
				Enabled:     false,
				Description: "test gate",
			},
		}})
	})
	assert.NotPanics(t, func() { WriteHTMLPageFooter(buf) })
	assert.NotPanics(t, func() { WriteHTMLPageFooter(buf) })
}
