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

package pipelines // import "go.opentelemetry.io/collector/service/internal/pipelines"

import (
	"context"
	"net/http"
	"sort"

	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/service/internal/builder"
	"go.opentelemetry.io/collector/service/internal/zpages"
)

const (
	zPipelineName  = "zpipelinename"
	zComponentName = "zcomponentname"
	zComponentKind = "zcomponentkind"
)

// Pipelines is set of all pipelines created from exporter configs.
type Pipelines struct {
	telemetry component.TelemetrySettings

	exporters *builder.BuiltExporters
	pipelines builder.BuiltPipelines
	receivers builder.Receivers
}

// Build builds all pipelines from config.
func Build(ctx context.Context, telemetry component.TelemetrySettings, buildInfo component.BuildInfo, cfg *config.Config, factories component.Factories) (*Pipelines, error) {
	exporters, err := builder.BuildExporters(ctx, telemetry, buildInfo, cfg, factories.Exporters)
	if err != nil {
		return nil, err
	}

	pipelines, err := builder.BuildPipelines(telemetry, buildInfo, cfg, exporters, factories.Processors)
	if err != nil {
		return nil, err
	}

	receivers, err := builder.BuildReceivers(telemetry, buildInfo, cfg, pipelines, factories.Receivers)
	if err != nil {
		return nil, err
	}

	return &Pipelines{
		telemetry: telemetry,
		receivers: receivers,
		exporters: exporters,
		pipelines: pipelines,
	}, nil
}

// StartAll starts all pipelines.
//
// Start with exporters, processors (in revers configured order), then receivers.
// This is important so that components that are earlier in the pipeline and reference components that are
// later in the pipeline do not start sending data to later components which are not yet started.
func (bps *Pipelines) StartAll(ctx context.Context, host component.Host) error {
	bps.telemetry.Logger.Info("Starting exporters...")
	if err := bps.exporters.StartAll(ctx, host); err != nil {
		return err
	}

	bps.telemetry.Logger.Info("Starting processors...")
	if err := bps.pipelines.StartProcessors(ctx, host); err != nil {
		return err
	}

	bps.telemetry.Logger.Info("Starting receivers...")
	if err := bps.receivers.StartAll(ctx, host); err != nil {
		return err
	}
	return nil
}

// ShutdownAll stops all pipelines.
//
// Shutdown order is the reverse of starting: receivers, processors, then exporters.
// This gives senders a chance to send all their data to a not "shutdown" component.
func (bps *Pipelines) ShutdownAll(ctx context.Context) error {
	var errs error
	bps.telemetry.Logger.Info("Stopping receivers...")
	errs = multierr.Append(errs, bps.receivers.ShutdownAll(ctx))

	bps.telemetry.Logger.Info("Stopping processors...")
	errs = multierr.Append(errs, bps.pipelines.ShutdownProcessors(ctx))

	bps.telemetry.Logger.Info("Stopping exporters...")
	errs = multierr.Append(errs, bps.exporters.ShutdownAll(ctx))

	return errs
}

func (bps *Pipelines) GetExporters() map[config.DataType]map[config.ComponentID]component.Exporter {
	return bps.exporters.ToMapByDataType()
}

func (bps *Pipelines) HandleZPages(w http.ResponseWriter, r *http.Request) {
	qValues := r.URL.Query()
	pipelineName := qValues.Get(zPipelineName)
	componentName := qValues.Get(zComponentName)
	componentKind := qValues.Get(zComponentKind)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	zpages.WriteHTMLPageHeader(w, zpages.HeaderData{Title: "Pipelines"})
	zpages.WriteHTMLPipelinesSummaryTable(w, bps.getPipelinesSummaryTableData())
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

func (bps *Pipelines) getPipelinesSummaryTableData() zpages.SummaryPipelinesTableData {
	sumData := zpages.SummaryPipelinesTableData{}
	sumData.Rows = make([]zpages.SummaryPipelinesTableRowData, 0, len(bps.pipelines))
	for c, p := range bps.pipelines {
		// TODO: Change the template to use ID.
		var recvs []string
		for _, bRecv := range p.Config.Receivers {
			recvs = append(recvs, bRecv.String())
		}
		var procs []string
		for _, bProc := range p.Config.Processors {
			procs = append(procs, bProc.String())
		}
		var exps []string
		for _, bExp := range p.Config.Exporters {
			exps = append(exps, bExp.String())
		}
		row := zpages.SummaryPipelinesTableRowData{
			FullName:    c.String(),
			InputType:   string(c.Type()),
			MutatesData: p.MutatesData,
			Receivers:   recvs,
			Processors:  procs,
			Exporters:   exps,
		}
		sumData.Rows = append(sumData.Rows, row)
	}

	sort.Slice(sumData.Rows, func(i, j int) bool {
		return sumData.Rows[i].FullName < sumData.Rows[j].FullName
	})
	return sumData
}
