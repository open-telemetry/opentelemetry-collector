// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package templates // import "go.opentelemetry.io/collector/extension/zpagesextension/internal/templates"

import (
	_ "embed"
	"html/template"
	"io"
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

	//go:embed templates/extensions_table.html
	extensionsTableBytes    []byte
	extensionsTableTemplate = parseTemplate("extensions_table", extensionsTableBytes)

	//go:embed templates/page_header.html
	headerBytes    []byte
	headerTemplate = parseTemplate("header", headerBytes)

	//go:embed templates/page_footer.html
	footerBytes    []byte
	footerTemplate = parseTemplate("footer", footerBytes)

	//go:embed templates/pipelines_table.html
	pipelinesTableBytes    []byte
	pipelinesTableTemplate = parseTemplate("pipelines_table", pipelinesTableBytes)

	//go:embed templates/properties_table.html
	propertiesTableBytes    []byte
	propertiesTableTemplate = parseTemplate("properties_table", propertiesTableBytes)

	//go:embed templates/features_table.html
	featuresTableBytes    []byte
	featuresTableTemplate = parseTemplate("features_table", featuresTableBytes)
)

func parseTemplate(name string, bytes []byte) *template.Template {
	return template.Must(template.New(name).Funcs(templateFunctions).Parse(string(bytes)))
}

// HeaderData contains data for the header template.
type HeaderData struct {
	Title string
}

// WriteHTMLPageHeader writes the header.
func WriteHTMLPageHeader(w io.Writer, hd HeaderData) error {
	return headerTemplate.Execute(w, hd)
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

// WriteHTMLExtensionsSummaryTable writes the summary table for one component type (receivers, processors, exporters).
// Id does not write the header or footer.
func WriteHTMLExtensionsSummaryTable(w io.Writer, spd SummaryExtensionsTableData) error {
	return extensionsTableTemplate.Execute(w, spd)
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
func WriteHTMLPipelinesSummaryTable(w io.Writer, spd SummaryPipelinesTableData) error {
	return pipelinesTableTemplate.Execute(w, spd)
}

// ComponentHeaderData contains data for component header template.
type ComponentHeaderData struct {
	Name              string
	ComponentEndpoint string
	Link              bool
}

// WriteHTMLComponentHeader writes the header for components.
func WriteHTMLComponentHeader(w io.Writer, chd ComponentHeaderData) error {
	return componentHeaderTemplate.Execute(w, chd)
}

// PropertiesTableData contains data for properties table template.
type PropertiesTableData struct {
	Name       string
	Properties [][2]string
}

// WriteHTMLPropertiesTable writes the HTML for properties table.
func WriteHTMLPropertiesTable(w io.Writer, chd PropertiesTableData) error {
	return propertiesTableTemplate.Execute(w, chd)
}

// WriteHTMLPageFooter writes the footer.
func WriteHTMLPageFooter(w io.Writer) error {
	return footerTemplate.Execute(w, nil)
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

// FeatureGateTableData contains data for feature gate table template.
type FeatureGateTableData struct {
	Rows []FeatureGateTableRowData
}

// FeatureGateTableRowData contains data for one row in feature gate table template.
type FeatureGateTableRowData struct {
	ID           string
	Enabled      bool
	Description  string
	Stage        string
	FromVersion  string
	ToVersion    string
	ReferenceURL string
}

// WriteHTMLFeaturesTable writes a table summarizing registered feature gates.
func WriteHTMLFeaturesTable(w io.Writer, ftd FeatureGateTableData) error {
	return featuresTableTemplate.Execute(w, ftd)
}
