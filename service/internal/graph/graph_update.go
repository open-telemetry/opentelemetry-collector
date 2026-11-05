// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package graph // import "go.opentelemetry.io/collector/service/internal/graph"

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"go.uber.org/zap"
	"gonum.org/v1/gonum/graph"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/service/internal/builders"
)

// UpdateRequest holds the receiver and processor changes to apply.
type UpdateRequest struct {
	// ChangedReceivers lists receivers whose config changed. Additions,
	// removals, and membership changes are derived from the pipeline configs.
	ChangedReceivers   map[component.ID]bool
	NewReceiverCfgs    map[component.ID]component.Config
	ReceiverFactories  map[component.Type]receiver.Factory
	ChangedProcessors  map[component.ID]bool
	NewProcessorCfgs   map[component.ID]component.Config
	ProcessorFactories map[component.Type]processor.Factory
}

// processorReload is the processor work for one pipeline.
type processorReload struct {
	pipelineID pipeline.ID
	pipe       *pipelineNodes

	// oldProcessors are shut down and removed.
	oldProcessors []*processorNode
	// newProcessors replace them, in pipeline order.
	newProcessors []*processorNode
	// keptProcessors are the unchanged trailing processors; they keep running.
	keptProcessors []graph.Node
}

// Update reloads receivers and processors in one pass, restarting each
// affected component once.
//
// Restarted: receivers that were added, removed, or whose config or pipeline
// membership changed; and, in each pipeline whose processors changed, the
// changed processors, those upstream of them, and the pipeline's receivers.
// Unchanged trailing processors, exporters, connectors, and other pipelines
// keep running.
//
// It returns false without modifying the graph if a full reload is required:
// the set of pipelines changed, or a pipeline with processor changes is fed by
// a connector. If it returns true with an error, the graph may be partially
// modified and must be discarded.
func (g *Graph) Update(ctx context.Context, set Settings, req UpdateRequest, host *Host) (bool, error) {
	if host == nil {
		return false, errors.New("host cannot be nil")
	}

	// Find the pipelines with processor changes.
	if len(set.PipelineConfigs) != len(g.pipelines) {
		return false, nil
	}
	var reloads []*processorReload
	affectedPipelines := make(map[pipeline.ID]bool)
	for pipelineID, pipelineCfg := range set.PipelineConfigs {
		pipe, ok := g.pipelines[pipelineID]
		if !ok {
			return false, nil
		}
		reload := g.identifyProcessorChanges(pipelineID, pipe, pipelineCfg.Processors, req.ChangedProcessors)
		if reload == nil {
			continue
		}
		// A connector keeps a reference to the pipeline's capabilities node,
		// which cannot be swapped while the connector runs.
		for _, node := range pipe.receivers {
			if _, isConnector := node.(*connectorNode); isConnector {
				return false, nil
			}
		}
		reloads = append(reloads, reload)
		affectedPipelines[pipelineID] = true
	}

	currentReceivers := make(map[int64]*receiverNode)
	currentPipelines := make(map[int64]map[pipeline.ID]bool)
	for pipelineID, pipe := range g.pipelines {
		for nodeID, node := range pipe.receivers {
			if rn, ok := node.(*receiverNode); ok {
				currentReceivers[nodeID] = rn
				if currentPipelines[nodeID] == nil {
					currentPipelines[nodeID] = make(map[pipeline.ID]bool)
				}
				currentPipelines[nodeID][pipelineID] = true
			}
		}
	}

	desiredPipelines := make(map[int64]map[pipeline.ID]bool)
	for pipelineID, pipelineCfg := range set.PipelineConfigs {
		for _, recvID := range pipelineCfg.Receivers {
			if set.ConnectorBuilder.IsConfigured(recvID) {
				continue
			}
			nodeID := newReceiverNode(pipelineID.Signal(), recvID).ID()
			if desiredPipelines[nodeID] == nil {
				desiredPipelines[nodeID] = make(map[pipeline.ID]bool)
			}
			desiredPipelines[nodeID][pipelineID] = true
		}
	}

	toRemove := make(map[int64]*receiverNode)  // present now, absent in desired
	toRebuild := make(map[int64]*receiverNode) // present in both, but changed or upstream of a processor change
	toAdd := make(map[int64]bool)              // absent now, present in desired
	for nodeID, rn := range currentReceivers {
		switch {
		case desiredPipelines[nodeID] == nil:
			toRemove[nodeID] = rn
		case !maps.Equal(currentPipelines[nodeID], desiredPipelines[nodeID]) ||
			req.ChangedReceivers[rn.componentID] ||
			feedsAffectedPipeline(currentPipelines[nodeID], affectedPipelines):
			toRebuild[nodeID] = rn
		}
	}
	for nodeID := range desiredPipelines {
		if _, exists := currentReceivers[nodeID]; !exists {
			toAdd[nodeID] = true
		}
	}

	if len(toRemove) == 0 && len(toRebuild) == 0 && len(toAdd) == 0 && len(reloads) == 0 {
		g.telemetry.Logger.Info("Partial reload: no receiver or processor changes detected")
		return true, nil
	}

	// Builders are immutable, so install new ones for the new configs.
	set.ReceiverBuilder = builders.NewReceiver(req.NewReceiverCfgs, req.ReceiverFactories)
	host.Receivers = set.ReceiverBuilder
	if len(reloads) > 0 {
		set.ProcessorBuilder = builders.NewProcessor(req.NewProcessorCfgs, req.ProcessorFactories)
		host.Processors = set.ProcessorBuilder
	}

	// Stop receivers first so nothing is sending, then the replaced processors.
	shutdownReceivers := make(map[int64]*receiverNode, len(toRemove)+len(toRebuild))
	maps.Copy(shutdownReceivers, toRemove)
	maps.Copy(shutdownReceivers, toRebuild)
	for _, nodeID := range sortedKeys(shutdownReceivers) {
		if err := g.shutdownReceiverNode(ctx, nodeID, shutdownReceivers[nodeID], host); err != nil {
			return true, err
		}
	}
	for _, reload := range reloads {
		for _, pn := range reload.oldProcessors {
			if err := g.shutdownProcessorNode(ctx, pn, host); err != nil {
				return true, err
			}
		}
	}

	for nodeID := range shutdownReceivers {
		g.componentGraph.RemoveNode(nodeID)
		delete(g.instanceIDs, nodeID)
	}
	for _, pipe := range g.pipelines {
		for nodeID := range shutdownReceivers {
			delete(pipe.receivers, nodeID)
		}
	}
	for _, reload := range reloads {
		g.replaceProcessorNodes(reload)
	}

	started := make(map[int64]*receiverNode)
	for pipelineID, pipelineCfg := range set.PipelineConfigs {
		pipe := g.pipelines[pipelineID]
		for _, recvID := range pipelineCfg.Receivers {
			if set.ConnectorBuilder.IsConfigured(recvID) {
				continue
			}
			nodeID := newReceiverNode(pipelineID.Signal(), recvID).ID()
			if !toAdd[nodeID] && toRebuild[nodeID] == nil {
				continue // unchanged receiver, skip
			}
			rn := g.createReceiver(pipelineID, recvID)
			pipe.receivers[rn.ID()] = rn
			g.componentGraph.SetEdge(g.componentGraph.NewEdge(rn, pipe.capabilitiesNode))
			started[rn.ID()] = rn
		}
	}

	// Build downstream first so each component gets its next consumer.
	for _, reload := range reloads {
		for _, pn := range slices.Backward(reload.newProcessors) {
			next := g.nextConsumers(pn.ID())[0]
			if err := pn.buildComponent(ctx, set.Telemetry, set.BuildInfo, set.ProcessorBuilder, next); err != nil {
				return true, err
			}
		}
		g.buildCapabilitiesNode(reload.pipe.capabilitiesNode)
	}
	for _, nodeID := range sortedKeys(started) {
		rn := started[nodeID]
		if err := rn.buildComponent(ctx, set.Telemetry, set.BuildInfo, set.ReceiverBuilder, g.nextConsumers(nodeID)); err != nil {
			return true, fmt.Errorf("failed to build receiver %q: %w", rn.componentID, err)
		}
	}

	// Start receivers last so data flows only once everything is up.
	for _, reload := range reloads {
		for _, pn := range slices.Backward(reload.newProcessors) {
			if err := g.startReloadedNode(ctx, pn.ID(), pn, host); err != nil {
				return true, err
			}
		}
	}
	for _, nodeID := range sortedKeys(started) {
		if err := g.startReloadedNode(ctx, nodeID, started[nodeID], host); err != nil {
			return true, err
		}
	}

	g.telemetry.Logger.Info("Partial reload completed successfully")
	return true, nil
}

// feedsAffectedPipeline reports whether any pipeline in pipelines is affected.
func feedsAffectedPipeline(pipelines, affected map[pipeline.ID]bool) bool {
	for pipelineID := range pipelines {
		if affected[pipelineID] {
			return true
		}
	}
	return false
}

// identifyProcessorChanges returns the processor work for a pipeline, or nil if
// it is unaffected. Unchanged trailing processors are kept; they only reference
// the fan-out node.
func (g *Graph) identifyProcessorChanges(pipelineID pipeline.ID, pipe *pipelineNodes,
	newProcIDs []component.ID, changedProcessors map[component.ID]bool,
) *processorReload {
	old := make([]*processorNode, len(pipe.processors))
	for i, node := range pipe.processors {
		old[i] = node.(*processorNode)
	}

	kept := 0
	for kept < len(old) && kept < len(newProcIDs) {
		oldNode := old[len(old)-1-kept]
		newID := newProcIDs[len(newProcIDs)-1-kept]
		if oldNode.componentID != newID || changedProcessors[newID] {
			break
		}
		kept++
	}
	if kept == len(old) && kept == len(newProcIDs) {
		return nil
	}

	reload := &processorReload{
		pipelineID:     pipelineID,
		pipe:           pipe,
		oldProcessors:  old[:len(old)-kept],
		keptProcessors: slices.Clone(pipe.processors[len(old)-kept:]),
	}
	for _, procID := range newProcIDs[:len(newProcIDs)-kept] {
		reload.newProcessors = append(reload.newProcessors, newProcessorNode(pipelineID, procID))
	}
	return reload
}

// replaceProcessorNodes swaps in the new processor nodes and rewires the pipeline.
func (g *Graph) replaceProcessorNodes(reload *processorReload) {
	for _, pn := range reload.oldProcessors {
		g.componentGraph.RemoveNode(pn.ID())
		delete(g.instanceIDs, pn.ID())
	}

	pipe := reload.pipe
	pipe.processors = pipe.processors[:0]
	for _, pn := range reload.newProcessors {
		g.componentGraph.AddNode(pn)
		g.instanceIDs[pn.ID()] = componentstatus.NewInstanceID(pn.componentID, component.KindProcessor, reload.pipelineID)
		pipe.processors = append(pipe.processors, pn)
	}
	pipe.processors = append(pipe.processors, reload.keptProcessors...)

	var from graph.Node = pipe.capabilitiesNode
	for _, node := range pipe.processors {
		g.componentGraph.SetEdge(g.componentGraph.NewEdge(from, node))
		from = node
	}
	g.componentGraph.SetEdge(g.componentGraph.NewEdge(from, pipe.fanOutNode))
}

func (g *Graph) shutdownProcessorNode(ctx context.Context, pn *processorNode, host *Host) error {
	instanceID := g.instanceIDs[pn.ID()]
	host.Reporter.ReportStatus(instanceID, componentstatus.NewEvent(componentstatus.StatusStopping))
	if err := pn.Shutdown(ctx); err != nil {
		host.Reporter.ReportStatus(instanceID, componentstatus.NewPermanentErrorEvent(err))
		return fmt.Errorf("failed to shutdown processor %q: %w", pn.componentID, err)
	}
	host.Reporter.ReportStatus(instanceID, componentstatus.NewEvent(componentstatus.StatusStopped))
	return nil
}

// startReloadedNode starts a rebuilt component and reports its status.
func (g *Graph) startReloadedNode(ctx context.Context, nodeID int64, comp component.Component, host *Host) error {
	instanceID := g.instanceIDs[nodeID]
	host.Reporter.ReportStatus(instanceID, componentstatus.NewEvent(componentstatus.StatusStarting))
	if err := comp.Start(ctx, &HostWrapper{Host: host, InstanceID: instanceID}); err != nil {
		host.Reporter.ReportStatus(instanceID, componentstatus.NewPermanentErrorEvent(err))
		g.telemetry.Logger.WithOptions(zap.AddStacktrace(zap.DPanicLevel)).
			Error("Failed to start component during partial reload",
				zap.Error(err),
				zap.String("type", instanceID.Kind().String()),
				zap.String("id", instanceID.ComponentID().String()),
			)
		return fmt.Errorf("failed to start %s %q: %w", strings.ToLower(instanceID.Kind().String()), instanceID.ComponentID(), err)
	}
	host.Reporter.ReportOKIfStarting(instanceID)
	return nil
}

// sortedKeys returns the keys of m in ascending order.
func sortedKeys[V any](m map[int64]V) []int64 {
	keys := make([]int64, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
