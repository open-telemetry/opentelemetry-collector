// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package zpages // import "go.opentelemetry.io/collector/service/internal/zpages"

import (
	_ "embed"
	"html/template"
	"io"
	"log"
)

var (
	templateFunctions = template.FuncMap{
		"even":     even,
		"getKey":   getKey,
		"getValue": getValue,
	}

	//go:embed templates/component_header.html
	componentHeaderBytes    []byte
	componentHeaderTemplate = parseTemplate("component_header", componentHeaderBytes)

	//go:embed templates/page_header.html
	headerBytes    []byte
	headerTemplate = parseTemplate("header", headerBytes)

	//go:embed templates/page_footer.html
	footerBytes    []byte
	footerTemplate = parseTemplate("footer", footerBytes)

	//go:embed templates/pipelines_table.html
	pipelinesTableBytes    []byte
	pipelinesTableTemplate = parseTemplate("pipelines_table", pipelinesTableBytes)
)

func parseTemplate(name string, bytes []byte) *template.Template {
	return template.Must(template.New(name).Funcs(templateFunctions).Parse(string(bytes)))
}

// HeaderData contains data for the header template.
type HeaderData struct {
	Title string
}

// WriteHTMLPageHeader writes the header.
func WriteHTMLPageHeader(w io.Writer, hd HeaderData) {
	if err := headerTemplate.Execute(w, hd); err != nil {
		log.Printf("zpages: executing template: %v", err)
	}
}

// SummaryExtensionsTableData contains data for extensions summary table template.
type SummaryExtensionsTableData struct {
	Rows []SummaryExtensionsTableRowData
}

// SummaryExtensionsTableRowData contains data for one row in extensions summary table template.
type SummaryExtensionsTableRowData struct {
	FullName string
	Enabled  bool
}

// SummaryPipelinesTableData contains data for pipelines summary table template.
type SummaryPipelinesTableData struct {
	Rows []SummaryPipelinesTableRowData
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

// WriteHTMLPageFooter writes the footer.
func WriteHTMLPageFooter(w io.Writer) {
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
