// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package zpagesextension // import "go.opentelemetry.io/collector/extension/zpagesextension"

import (
	"net/http"
	"path"
	"runtime"
	"sort"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/zpagesextension/internal/templates"
	"go.opentelemetry.io/collector/featuregate"
)

const (
	// Paths
	zServicePath   = "servicez"
	zPipelinePath  = "pipelinez"
	zExtensionPath = "extensionz"
	zFeaturePath   = "featurez"
	zExtensionName = "zextensionname"
	// URL Params
	zPipelineName  = "pipelinenamez"
	zComponentName = "componentnamez"
	zComponentKind = "componentkindz"
)

var (
	// InfoVar is a singleton instance of the Info struct.
	runtimeInfoVar [][2]string
)

func init() {
	runtimeInfoVar = [][2]string{
		{"StartTimestamp", time.Now().String()},
		{"Go", runtime.Version()},
		{"OS", runtime.GOOS},
		{"Arch", runtime.GOARCH},
		// Add other valuable runtime information here.
	}
}

func registerZPages(mux *http.ServeMux, pathPrefix string, host component.Host, cs extension.CreateSettings) {
	handler := &zpagesHandler{
		createSettings: cs,
		host:           host,
	}
	mux.HandleFunc(path.Join(pathPrefix, zServicePath), handler.zPagesRequest)
	mux.HandleFunc(path.Join(pathPrefix, zExtensionPath), handler.handleServiceExtensions)
	mux.HandleFunc(path.Join(pathPrefix, zFeaturePath), handler.handleFeaturezRequest)
	mux.HandleFunc(path.Join(pathPrefix, zPipelinePath), handler.handlePipelinezRequest)
}

type zpagesHandler struct {
	createSettings extension.CreateSettings
	host           component.Host
}

func (zh *zpagesHandler) handleServiceExtensions(w http.ResponseWriter, r *http.Request) {
	extensionName := r.URL.Query().Get(zExtensionName)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.WriteHTMLPageHeader(w, templates.HeaderData{Title: "Extensions"})
	data := templates.SummaryExtensionsTableData{}

	data.Rows = make([]templates.SummaryExtensionsTableRowData, 0, len(zh.host.GetExtensions()))
	for id := range zh.host.GetExtensions() {
		row := templates.SummaryExtensionsTableRowData{FullName: id.String()}
		data.Rows = append(data.Rows, row)
	}

	sort.Slice(data.Rows, func(i, j int) bool {
		return data.Rows[i].FullName < data.Rows[j].FullName
	})
	templates.WriteHTMLExtensionsSummaryTable(w, data)
	if extensionName != "" {
		templates.WriteHTMLComponentHeader(w, templates.ComponentHeaderData{
			Name: extensionName,
		})
		// TODO: Add config + status info.
	}
	templates.WriteHTMLPageFooter(w)
}

func (zh *zpagesHandler) zPagesRequest(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.WriteHTMLPageHeader(w, templates.HeaderData{Title: "Service " + zh.createSettings.BuildInfo.Command})
	templates.WriteHTMLPropertiesTable(w, templates.PropertiesTableData{Name: "Build Info", Properties: getBuildInfoProperties(zh.createSettings.BuildInfo)})
	templates.WriteHTMLPropertiesTable(w, templates.PropertiesTableData{Name: "Runtime Info", Properties: runtimeInfoVar})
	templates.WriteHTMLComponentHeader(w, templates.ComponentHeaderData{
		Name:              "Pipelines",
		ComponentEndpoint: zPipelinePath,
		Link:              true,
	})
	templates.WriteHTMLComponentHeader(w, templates.ComponentHeaderData{
		Name:              "Extensions",
		ComponentEndpoint: zExtensionPath,
		Link:              true,
	})
	templates.WriteHTMLComponentHeader(w, templates.ComponentHeaderData{
		Name:              "Features",
		ComponentEndpoint: zFeaturePath,
		Link:              true,
	})
	templates.WriteHTMLPageFooter(w)
}

func (zh *zpagesHandler) handleFeaturezRequest(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.WriteHTMLPageHeader(w, templates.HeaderData{Title: "Feature Gates"})
	templates.WriteHTMLFeaturesTable(w, getFeaturesTableData())
	templates.WriteHTMLPageFooter(w)
}

func getFeaturesTableData() templates.FeatureGateTableData {
	data := templates.FeatureGateTableData{}
	featuregate.GlobalRegistry().VisitAll(func(gate *featuregate.Gate) {
		data.Rows = append(data.Rows, templates.FeatureGateTableRowData{
			ID:           gate.ID(),
			Enabled:      gate.IsEnabled(),
			Description:  gate.Description(),
			Stage:        gate.Stage().String(),
			FromVersion:  gate.FromVersion(),
			ToVersion:    gate.ToVersion(),
			ReferenceURL: gate.ReferenceURL(),
		})
	})
	return data
}

func getBuildInfoProperties(buildInfo component.BuildInfo) [][2]string {
	return [][2]string{
		{"Command", buildInfo.Command},
		{"Description", buildInfo.Description},
		{"Version", buildInfo.Version},
	}
}

func (zh *zpagesHandler) handlePipelinezRequest(w http.ResponseWriter, r *http.Request) {
	qValues := r.URL.Query()
	pipelineName := qValues.Get(zPipelineName)
	componentName := qValues.Get(zComponentName)
	componentKind := qValues.Get(zComponentKind)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.WriteHTMLPageHeader(w, templates.HeaderData{Title: "builtPipelines"})

	sumData := templates.SummaryPipelinesTableData{}
	if ghost, ok := zh.host.(interface {
		GetGraph() []struct {
			FullName    string
			InputType   string
			MutatesData bool
			Receivers   []string
			Processors  []string
			Exporters   []string
		}
	}); ok {
		g := ghost.GetGraph()
		sumData.Rows = make([]templates.SummaryPipelinesTableRowData, 0, len(g))
		for _, p := range g {
			sumData.Rows = append(sumData.Rows, templates.SummaryPipelinesTableRowData{
				FullName:    p.FullName,
				InputType:   p.InputType,
				MutatesData: p.MutatesData,
				Receivers:   p.Receivers,
				Processors:  p.Processors,
				Exporters:   p.Exporters,
			})
		}
	}

	sort.Slice(sumData.Rows, func(i, j int) bool {
		return sumData.Rows[i].FullName < sumData.Rows[j].FullName
	})
	templates.WriteHTMLPipelinesSummaryTable(w, sumData)

	if pipelineName != "" && componentName != "" && componentKind != "" {
		fullName := componentName
		if componentKind == "processor" {
			fullName = pipelineName + "/" + componentName
		}
		templates.WriteHTMLComponentHeader(w, templates.ComponentHeaderData{
			Name: componentKind + ": " + fullName,
		})
		// TODO: Add config + status info.
	}
	templates.WriteHTMLPageFooter(w)
}
