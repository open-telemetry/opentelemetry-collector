// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package templates

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

func TestNoError(t *testing.T) {
	buf := new(bytes.Buffer)
	assert.NoError(t, WriteHTMLPageHeader(buf, HeaderData{Title: "Foo"}))
	assert.NoError(t, WriteHTMLComponentHeader(buf, ComponentHeaderData{Name: "Bar"}))
	assert.NoError(t, WriteHTMLComponentHeader(buf, ComponentHeaderData{Name: "Bar", ComponentEndpoint: "pagez", Link: true}))
	assert.NoError(t, WriteHTMLPipelinesSummaryTable(buf, SummaryPipelinesTableData{
		Rows: []SummaryPipelinesTableRowData{{
			FullName:    "test",
			InputType:   "metrics",
			MutatesData: false,
			Receivers:   []string{"oc"},
			Processors:  []string{"nop"},
			Exporters:   []string{"oc"},
		}},
	}))
	assert.NoError(t,
		WriteHTMLExtensionsSummaryTable(buf, SummaryExtensionsTableData{
			Rows: []SummaryExtensionsTableRowData{{
				FullName: "test",
			}},
		}))
	assert.NoError(t,
		WriteHTMLPropertiesTable(buf, PropertiesTableData{Name: "Bar", Properties: [][2]string{{"key", "value"}}}))
	assert.NoError(t, WriteHTMLFeaturesTable(buf, FeatureGateTableData{Rows: []FeatureGateTableRowData{
		{
			ID:          "test",
			Enabled:     false,
			Description: "test gate",
		},
	}}))
	assert.NoError(t, WriteHTMLPageFooter(buf))
	assert.NoError(t, WriteHTMLPageFooter(buf))
}
