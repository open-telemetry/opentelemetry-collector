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

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/service/internal/runtimeinfo"
	"go.opentelemetry.io/collector/service/internal/zpages"
)

const (
	// Paths
	zServicePath   = "servicez"
	zPipelinePath  = "pipelinez"
	zExtensionPath = "extensionz"
	zFeaturePath   = "featurez"
)

func (host *serviceHost) RegisterZPages(mux *http.ServeMux, pathPrefix string) {
	mux.HandleFunc(path.Join(pathPrefix, zServicePath), host.zPagesRequest)
	mux.HandleFunc(path.Join(pathPrefix, zPipelinePath), host.pipelines.HandleZPages)
	mux.HandleFunc(path.Join(pathPrefix, zExtensionPath), host.serviceExtensions.HandleZPages)
	mux.HandleFunc(path.Join(pathPrefix, zFeaturePath), handleFeaturezRequest)
}

func (host *serviceHost) zPagesRequest(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	zpages.WriteHTMLPageHeader(w, zpages.HeaderData{Title: "Service " + host.buildInfo.Command})
	zpages.WriteHTMLPropertiesTable(w, zpages.PropertiesTableData{Name: "Build Info", Properties: getBuildInfoProperties(host.buildInfo)})
	zpages.WriteHTMLPropertiesTable(w, zpages.PropertiesTableData{Name: "Runtime Info", Properties: runtimeinfo.Info()})
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

func handleFeaturezRequest(w http.ResponseWriter, _ *http.Request) {
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
