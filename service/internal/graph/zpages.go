// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package graph // import "go.opentelemetry.io/collector/service/internal/graph"

import (
	"net/http"
	"sort"

	"go.opentelemetry.io/collector/service/internal/zpages"
)

const (

	// URL Params
	zPipelineName  = "pipelinenamez"
	zComponentName = "componentnamez"
	zComponentKind = "componentkindz"
)

func (g *Graph) HandleZPages(w http.ResponseWriter, r *http.Request) {
	qValues := r.URL.Query()
	pipelineName := qValues.Get(zPipelineName)
	componentName := qValues.Get(zComponentName)
	componentKind := qValues.Get(zComponentKind)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	zpages.WriteHTMLPageHeader(w, zpages.HeaderData{Title: "builtPipelines"})

	sumData := zpages.SummaryPipelinesTableData{}
	sumData.Rows = make([]zpages.SummaryPipelinesTableRowData, 0, len(g.pipelines))
	for c, p := range g.pipelines {
		recvIDs := make([]string, 0, len(p.receivers))
		for _, c := range p.receivers {
			switch n := c.(type) {
			case *receiverNode:
				recvIDs = append(recvIDs, n.componentID.String())
			case *connectorNode:
				recvIDs = append(recvIDs, n.componentID.String()+" (connector)")
			}
		}
		procIDs := make([]string, 0, len(p.processors))
		for _, c := range p.processors {
			procIDs = append(procIDs, c.(*processorNode).componentID.String())
		}
		exprIDs := make([]string, 0, len(p.exporters))
		for _, c := range p.exporters {
			switch n := c.(type) {
			case *exporterNode:
				exprIDs = append(exprIDs, n.componentID.String())
			case *connectorNode:
				exprIDs = append(exprIDs, n.componentID.String()+" (connector)")
			}
		}

		sumData.Rows = append(sumData.Rows, zpages.SummaryPipelinesTableRowData{
			FullName:    c.String(),
			InputType:   c.Signal().String(),
			MutatesData: p.capabilitiesNode.getConsumer().Capabilities().MutatesData,
			Receivers:   recvIDs,
			Processors:  procIDs,
			Exporters:   exprIDs,
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
