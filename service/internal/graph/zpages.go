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
			procIDs = append(procIDs, c.componentID.String())
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
			InputType:   string(c.Type()),
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
