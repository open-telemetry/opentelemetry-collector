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

package service // import "go.opentelemetry.io/collector/service"

import (
	"net/http"
	"path"
	"sort"

	otelzpages "go.opentelemetry.io/contrib/zpages"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/internal/version"
	"go.opentelemetry.io/collector/service/internal/zpages"
)

const (
	tracezPath     = "tracez"
	servicezPath   = "servicez"
	pipelinezPath  = "pipelinez"
	extensionzPath = "extensionz"

	zPipelineName  = "zpipelinename"
	zComponentName = "zcomponentname"
	zComponentKind = "zcomponentkind"
	zExtensionName = "zextensionname"
)

func (srv *service) RegisterZPages(mux *http.ServeMux, pathPrefix string) {
	mux.Handle(path.Join(pathPrefix, tracezPath), otelzpages.NewTracezHandler(srv.zPagesSpanProcessor))
	mux.HandleFunc(path.Join(pathPrefix, servicezPath), srv.handleServicezRequest)
	mux.HandleFunc(path.Join(pathPrefix, pipelinezPath), srv.handlePipelinezRequest)
	mux.HandleFunc(path.Join(pathPrefix, extensionzPath), func(w http.ResponseWriter, r *http.Request) {
		handleExtensionzRequest(srv, w, r)
	})
}

func (srv *service) handleServicezRequest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	zpages.WriteHTMLPageHeader(w, zpages.HeaderData{Title: "service"})
	zpages.WriteHTMLComponentHeader(w, zpages.ComponentHeaderData{
		Name:              "Pipelines",
		ComponentEndpoint: pipelinezPath,
		Link:              true,
	})
	zpages.WriteHTMLComponentHeader(w, zpages.ComponentHeaderData{
		Name:              "Extensions",
		ComponentEndpoint: extensionzPath,
		Link:              true,
	})
	zpages.WriteHTMLPropertiesTable(w, zpages.PropertiesTableData{Name: "Build And Runtime", Properties: version.RuntimeVar()})
	zpages.WriteHTMLPageFooter(w)
}

func (srv *service) handlePipelinezRequest(w http.ResponseWriter, r *http.Request) {
	qValues := r.URL.Query()
	pipelineName := qValues.Get(zPipelineName)
	componentName := qValues.Get(zComponentName)
	componentKind := qValues.Get(zComponentKind)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	zpages.WriteHTMLPageHeader(w, zpages.HeaderData{Title: "Pipelines"})
	zpages.WriteHTMLPipelinesSummaryTable(w, srv.getPipelinesSummaryTableData())
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

func (srv *service) getPipelinesSummaryTableData() zpages.SummaryPipelinesTableData {
	data := zpages.SummaryPipelinesTableData{}

	data.Rows = make([]zpages.SummaryPipelinesTableRowData, 0, len(srv.builtPipelines))
	for c, p := range srv.builtPipelines {
		// TODO: Change the template to use ID.
		var recvs []string
		for _, recvID := range p.Config.Receivers {
			recvs = append(recvs, recvID.String())
		}
		var procs []string
		for _, procID := range p.Config.Processors {
			procs = append(procs, procID.String())
		}
		var exps []string
		for _, expID := range p.Config.Exporters {
			exps = append(exps, expID.String())
		}
		row := zpages.SummaryPipelinesTableRowData{
			FullName:    c.String(),
			InputType:   string(c.Type()),
			MutatesData: p.MutatesData,
			Receivers:   recvs,
			Processors:  procs,
			Exporters:   exps,
		}
		data.Rows = append(data.Rows, row)
	}

	sort.Slice(data.Rows, func(i, j int) bool {
		return data.Rows[i].FullName < data.Rows[j].FullName
	})
	return data
}

func handleExtensionzRequest(host component.Host, w http.ResponseWriter, r *http.Request) {
	extensionName := r.URL.Query().Get(zExtensionName)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	zpages.WriteHTMLPageHeader(w, zpages.HeaderData{Title: "Extensions"})
	zpages.WriteHTMLExtensionsSummaryTable(w, getExtensionsSummaryTableData(host))
	if extensionName != "" {
		zpages.WriteHTMLComponentHeader(w, zpages.ComponentHeaderData{
			Name: extensionName,
		})
		// TODO: Add config + status info.
	}
	zpages.WriteHTMLPageFooter(w)
}

func getExtensionsSummaryTableData(host component.Host) zpages.SummaryExtensionsTableData {
	data := zpages.SummaryExtensionsTableData{}

	extensions := host.GetExtensions()
	data.Rows = make([]zpages.SummaryExtensionsTableRowData, 0, len(extensions))
	for c := range extensions {
		row := zpages.SummaryExtensionsTableRowData{FullName: c.String()}
		data.Rows = append(data.Rows, row)
	}

	sort.Slice(data.Rows, func(i, j int) bool {
		return data.Rows[i].FullName < data.Rows[j].FullName
	})
	return data
}
