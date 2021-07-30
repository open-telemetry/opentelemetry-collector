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
	r.ParseForm() // nolint:errcheck
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	zpages.WriteHTMLHeader(w, zpages.HeaderData{Title: "service"})
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
	zpages.WriteHTMLFooter(w)
}

func (srv *service) handlePipelinezRequest(w http.ResponseWriter, r *http.Request) {
	r.ParseForm() // nolint:errcheck
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	pipelineName := r.Form.Get(zPipelineName)
	componentName := r.Form.Get(zComponentName)
	componentKind := r.Form.Get(zComponentKind)
	zpages.WriteHTMLHeader(w, zpages.HeaderData{Title: "Pipelines"})
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
	zpages.WriteHTMLFooter(w)
}

func (srv *service) getPipelinesSummaryTableData() zpages.SummaryPipelinesTableData {
	data := zpages.SummaryPipelinesTableData{
		ComponentEndpoint: pipelinezPath,
	}

	data.Rows = make([]zpages.SummaryPipelinesTableRowData, 0, len(srv.builtPipelines))
	for c, p := range srv.builtPipelines {
		// TODO: Change the template to use ID.
		var recvs []string
		for _, recvID := range c.Receivers {
			recvs = append(recvs, recvID.String())
		}
		var procs []string
		for _, procID := range c.Processors {
			procs = append(procs, procID.String())
		}
		var exps []string
		for _, expID := range c.Exporters {
			exps = append(exps, expID.String())
		}
		row := zpages.SummaryPipelinesTableRowData{
			FullName:    c.Name,
			InputType:   string(c.InputType),
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
	r.ParseForm() // nolint:errcheck
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	extensionName := r.Form.Get(zExtensionName)
	zpages.WriteHTMLHeader(w, zpages.HeaderData{Title: "Extensions"})
	zpages.WriteHTMLExtensionsSummaryTable(w, getExtensionsSummaryTableData(host))
	if extensionName != "" {
		zpages.WriteHTMLComponentHeader(w, zpages.ComponentHeaderData{
			Name: extensionName,
		})
		// TODO: Add config + status info.
	}
	zpages.WriteHTMLFooter(w)
}

func getExtensionsSummaryTableData(host component.Host) zpages.SummaryExtensionsTableData {
	data := zpages.SummaryExtensionsTableData{
		ComponentEndpoint: extensionzPath,
	}

	extensions := host.GetExtensions()
	data.Rows = make([]zpages.SummaryExtensionsTableRowData, 0, len(extensions))
	for c := range extensions {
		row := zpages.SummaryExtensionsTableRowData{FullName: c.Name()}
		data.Rows = append(data.Rows, row)
	}

	sort.Slice(data.Rows, func(i, j int) bool {
		return data.Rows[i].FullName < data.Rows[j].FullName
	})
	return data
}
