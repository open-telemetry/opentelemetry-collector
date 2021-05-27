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
	"html/template"
	"io"
	"io/ioutil"
	"log"

	"go.opentelemetry.io/collector/service/internal/zpages/tmplgen"
)

var (
	fs                = tmplgen.FS(false)
	templateFunctions = template.FuncMap{
		"even":     even,
		"getKey":   getKey,
		"getValue": getValue,
	}
	componentHeaderTemplate = parseTemplate("component_header")
	extensionsTableTemplate = parseTemplate("extensions_table")
	headerTemplate          = parseTemplate("header")
	footerTemplate          = parseTemplate("footer")
	pipelinesTableTemplate  = parseTemplate("pipelines_table")
	propertiesTableTemplate = parseTemplate("properties_table")
)

func parseTemplate(name string) *template.Template {
	f, err := fs.Open("/templates/" + name + ".html")
	if err != nil {
		log.Panicf("%v: %v", name, err)
	}
	defer f.Close()
	text, err := ioutil.ReadAll(f)
	if err != nil {
		log.Panicf("%v: %v", name, err)
	}
	return template.Must(template.New(name).Funcs(templateFunctions).Parse(string(text)))
}

// HeaderData contains data for the header template.
type HeaderData struct {
	Title string
}

// WriteHTMLHeader writes the header.
func WriteHTMLHeader(w io.Writer, hd HeaderData) {
	if err := headerTemplate.Execute(w, hd); err != nil {
		log.Printf("zpages: executing template: %v", err)
	}
}

// SummaryExtensionsTableData contains data for extensions summary table template.
type SummaryExtensionsTableData struct {
	ComponentEndpoint string
	Rows              []SummaryExtensionsTableRowData
}

// SummaryExtensionsTableRowData contains data for one row in extensions summary table template.
type SummaryExtensionsTableRowData struct {
	FullName string
	Enabled  bool
}

// WriteHTMLExtensionsSummaryTable writes the summary table for one component type (receivers, processors, exporters).
// Id does not write the header or footer.
func WriteHTMLExtensionsSummaryTable(w io.Writer, spd SummaryExtensionsTableData) {
	if err := extensionsTableTemplate.Execute(w, spd); err != nil {
		log.Printf("zpages: executing template: %v", err)
	}
}

// SummaryPipelinesTableData contains data for pipelines summary table template.
type SummaryPipelinesTableData struct {
	ComponentEndpoint string
	Rows              []SummaryPipelinesTableRowData
}

// SummaryPipelinesTableRowData contains data for one row in pipelines summary table template.
type SummaryPipelinesTableRowData struct {
	FullName    string
	InputType   string
	MutatesData bool
	Receivers   []string
	Processors  []string
	Exporters   []string
}

// WriteHTMLPipelinesSummaryTable writes the summary table for one component type (receivers, processors, exporters).
// Id does not write the header or footer.
func WriteHTMLPipelinesSummaryTable(w io.Writer, spd SummaryPipelinesTableData) {
	if err := pipelinesTableTemplate.Execute(w, spd); err != nil {
		log.Printf("zpages: executing template: %v", err)
	}
}

// ComponentHeaderData contains data for component header template.
type ComponentHeaderData struct {
	Name              string
	ComponentEndpoint string
	Link              bool
}

// WriteHTMLComponentHeader writes the header for components.
func WriteHTMLComponentHeader(w io.Writer, chd ComponentHeaderData) {
	if err := componentHeaderTemplate.Execute(w, chd); err != nil {
		log.Printf("zpages: executing template: %v", err)
	}
}

// PropertiesTableData contains data for properties table template.
type PropertiesTableData struct {
	Name       string
	Properties [][2]string
}

// WriteHTMLPropertiesTable writes the HTML for properties table.
func WriteHTMLPropertiesTable(w io.Writer, chd PropertiesTableData) {
	if err := propertiesTableTemplate.Execute(w, chd); err != nil {
		log.Printf("zpages: executing template: %v", err)
	}
}

// WriteHTMLFooter writes the footer.
func WriteHTMLFooter(w io.Writer) {
	if err := footerTemplate.Execute(w, nil); err != nil {
		log.Printf("zpages: executing template: %v", err)
	}
}

func even(x int) bool {
	return x%2 == 0
}

func getKey(row [2]string) string {
	return row[0]
}

func getValue(row [2]string) string {
	return row[1]
}
