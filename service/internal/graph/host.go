// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package graph // import "go.opentelemetry.io/collector/service/internal/graph"

import (
	"net/http"
	"path"
	"runtime"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/service/extensions"
	"go.opentelemetry.io/collector/service/hostcapabilities"
	"go.opentelemetry.io/collector/service/internal/builders"
	"go.opentelemetry.io/collector/service/internal/moduleinfo"
	"go.opentelemetry.io/collector/service/internal/status"
	"go.opentelemetry.io/collector/service/internal/zpages"
)

var (
	_ component.Host                    = (*Host)(nil)
	_ hostcapabilities.ModuleInfo       = (*Host)(nil)
	_ hostcapabilities.ExposeExporters  = (*Host)(nil)
	_ hostcapabilities.ComponentFactory = (*Host)(nil)
)

type Host struct {
	AsyncErrorChannel chan error
	Receivers         *builders.ReceiverBuilder
	Processors        *builders.ProcessorBuilder
	Exporters         *builders.ExporterBuilder
	Connectors        *builders.ConnectorBuilder
	Extensions        *builders.ExtensionBuilder

	ModuleInfos moduleinfo.ModuleInfos
	BuildInfo   component.BuildInfo

	Pipelines         *Graph
	ServiceExtensions *extensions.Extensions

	Reporter status.Reporter
}

func (host *Host) GetFactory(kind component.Kind, componentType component.Type) component.Factory {
	switch kind {
	case component.KindReceiver:
		return host.Receivers.Factory(componentType)
	case component.KindProcessor:
		return host.Processors.Factory(componentType)
	case component.KindExporter:
		return host.Exporters.Factory(componentType)
	case component.KindConnector:
		return host.Connectors.Factory(componentType)
	case component.KindExtension:
		return host.Extensions.Factory(componentType)
	}
	return nil
}

func (host *Host) GetExtensions() map[component.ID]component.Component {
	return host.ServiceExtensions.GetExtensions()
}

func (host *Host) GetModuleInfos() moduleinfo.ModuleInfos {
	return host.ModuleInfos
}

// Deprecated: [0.79.0] This function will be removed in the future.
// Several components in the contrib repository use this function so it cannot be removed
// before those cases are removed. In most cases, use of this function can be replaced by a
// connector. See https://github.com/open-telemetry/opentelemetry-collector/issues/7370 and
// https://github.com/open-telemetry/opentelemetry-collector/pull/7390#issuecomment-1483710184
// for additional information.
func (host *Host) GetExporters() map[pipeline.Signal]map[component.ID]component.Component {
	return host.Pipelines.GetExporters()
}

func (host *Host) NotifyComponentStatusChange(source *componentstatus.InstanceID, event *componentstatus.Event) {
	host.ServiceExtensions.NotifyComponentStatusChange(source, event)
	if event.Status() == componentstatus.StatusFatalError {
		host.AsyncErrorChannel <- event.Err()
	}
}

const (
	// Paths
	zServicePath   = "servicez"
	zPipelinePath  = "pipelinez"
	zExtensionPath = "extensionz"
	zFeaturePath   = "featurez"
)

// InfoVar is a singleton instance of the Info struct.
var runtimeInfoVar [][2]string

func init() {
	runtimeInfoVar = [][2]string{
		{"StartTimestamp", time.Now().String()},
		{"Go", runtime.Version()},
		{"OS", runtime.GOOS},
		{"Arch", runtime.GOARCH},
		// Add other valuable runtime information here.
	}
}

func (host *Host) RegisterZPages(mux *http.ServeMux, pathPrefix string) {
	mux.HandleFunc(path.Join(pathPrefix, zServicePath), host.zPagesRequest)
	mux.HandleFunc(path.Join(pathPrefix, zPipelinePath), host.Pipelines.HandleZPages)
	mux.HandleFunc(path.Join(pathPrefix, zExtensionPath), host.ServiceExtensions.HandleZPages)
	mux.HandleFunc(path.Join(pathPrefix, zFeaturePath), handleFeaturezRequest)
}

func (host *Host) zPagesRequest(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	zpages.WriteHTMLPageHeader(w, zpages.HeaderData{Title: "Service " + host.BuildInfo.Command})
	zpages.WriteHTMLPropertiesTable(w, zpages.PropertiesTableData{Name: "Build Info", Properties: getBuildInfoProperties(host.BuildInfo)})
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
