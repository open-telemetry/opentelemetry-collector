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
	"go.opentelemetry.io/collector/extension/zpagesextension/internal/zpages"
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
	zpages.WriteHTMLPageHeader(w, zpages.HeaderData{Title: "Extensions"})
	data := zpages.SummaryExtensionsTableData{}

	data.Rows = make([]zpages.SummaryExtensionsTableRowData, 0, len(zh.host.GetExtensions()))
	for id := range zh.host.GetExtensions() {
		row := zpages.SummaryExtensionsTableRowData{FullName: id.String()}
		data.Rows = append(data.Rows, row)
	}

	sort.Slice(data.Rows, func(i, j int) bool {
		return data.Rows[i].FullName < data.Rows[j].FullName
	})
	zpages.WriteHTMLExtensionsSummaryTable(w, data)
	if extensionName != "" {
		zpages.WriteHTMLComponentHeader(w, zpages.ComponentHeaderData{
			Name: extensionName,
		})
		// TODO: Add config + status info.
	}
	zpages.WriteHTMLPageFooter(w)
}

func (zh *zpagesHandler) zPagesRequest(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	zpages.WriteHTMLPageHeader(w, zpages.HeaderData{Title: "Service " + zh.createSettings.BuildInfo.Command})
	zpages.WriteHTMLPropertiesTable(w, zpages.PropertiesTableData{Name: "Build Info", Properties: getBuildInfoProperties(zh.createSettings.BuildInfo)})
	zpages.WriteHTMLPropertiesTable(w, zpages.PropertiesTableData{Name: "Runtime Info", Properties: runtimeInfoVar})
	zpages.WriteHTMLComponentHeader(w, zpages.ComponentHeaderData{
		Name:              "Pipelines",
		ComponentEndpoint: zPipelinePath,
		Link:              true,
	})
	zpages.WriteHTMLComponentHeader(w, zpages.ComponentHeaderData{
		Name:              "Extensions",
		ComponentEndpoint: zExtensionPath,
		Link:              true,
	})
	zpages.WriteHTMLComponentHeader(w, zpages.ComponentHeaderData{
		Name:              "Features",
		ComponentEndpoint: zFeaturePath,
		Link:              true,
	})
	zpages.WriteHTMLPageFooter(w)
}

func (zh *zpagesHandler) handleFeaturezRequest(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	zpages.WriteHTMLPageHeader(w, zpages.HeaderData{Title: "Feature Gates"})
	zpages.WriteHTMLFeaturesTable(w, getFeaturesTableData())
	zpages.WriteHTMLPageFooter(w)
}

func getFeaturesTableData() zpages.FeatureGateTableData {
	data := zpages.FeatureGateTableData{}
	featuregate.GlobalRegistry().VisitAll(func(gate *featuregate.Gate) {
		data.Rows = append(data.Rows, zpages.FeatureGateTableRowData{
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
	zpages.WriteHTMLPageHeader(w, zpages.HeaderData{Title: "builtPipelines"})

	sumData := zpages.SummaryPipelinesTableData{}
	g := zh.host.GetGraph()
	sumData.Rows = make([]zpages.SummaryPipelinesTableRowData, 0, len(g.Pipelines))
	for _, p := range g.Pipelines {
		sumData.Rows = append(sumData.Rows, zpages.SummaryPipelinesTableRowData{
			FullName:    p.FullName,
			InputType:   p.InputType,
			MutatesData: p.MutatesData,
			Receivers:   p.Receivers,
			Processors:  p.Processors,
			Exporters:   p.Exporters,
		})
	}
	sort.Slice(sumData.Rows, func(i, j int) bool {
		return sumData.Rows[i].FullName < sumData.Rows[j].FullName
	})
	zpages.WriteHTMLPipelinesSummaryTable(w, sumData)

	if pipelineName != "" && componentName != "" && componentKind != "" {
		fullName := componentName
		if componentKind == "processor" {
			fullName = pipelineName + "/" + componentName
		}
		zpages.WriteHTMLComponentHeader(w, zpages.ComponentHeaderData{
			Name: componentKind + ": " + fullName,
		})
		// TODO: Add config + status info.
	}
	zpages.WriteHTMLPageFooter(w)
}
