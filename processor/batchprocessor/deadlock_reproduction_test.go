// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package batchprocessor

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// TestDeadlockReproductionUsingExporterHelper reproduces the deadlock that occurs when using WaitForResult: true
// with the exporterhelper system, matching how the batch processor actually works.
func TestDeadlockReproductionUsingExporterHelper(t *testing.T) {
	t.Log("Starting ExporterHelper deadlock reproduction test...")

	// Create a consumer function that will be called by the exporter
	var consumerFunc consumer.ConsumeTracesFunc = func(ctx context.Context, td ptrace.Traces) error {
		t.Log("Consumer function received traces")
		time.Sleep(100 * time.Millisecond) // Simulate processing time
		t.Log("Consumer function completed processing")
		return nil
	}

	// Create configuration that matches what the batch processor uses
	config := struct{}{}

	// Create QueueBatch config that matches the batch processor configuration with WaitForResult: true
	queueBatchConfig := translateToExporterHelperConfig(&Config{
		SendBatchSize:    1, // Batch immediately for testing
		SendBatchMaxSize: 10,
		Timeout:          1 * time.Second,
	})
	queueBatchConfig.WaitForResult = true // Force this to true to reproduce the deadlock

	// Create exporter settings
	exporterSet := exporter.Settings{
		ID:                component.NewID(component.MustNewType("test")),
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
		BuildInfo:         component.NewDefaultBuildInfo(),
	}

	// Create the exporter using the same method as the batch processor
	exp, err := exporterhelper.NewTraces(
		context.Background(),
		exporterSet,
		config,
		consumerFunc,
		exporterhelper.WithQueue(queueBatchConfig),
	)
	if err != nil {
		t.Fatalf("Failed to create exporter: %v", err)
	}

	ctx := context.Background()

	t.Log("Starting exporter...")
	err = exp.Start(ctx, component.Host(nil))
	if err != nil {
		t.Fatalf("Failed to start exporter: %v", err)
	}

	// Create test traces
	traces := ptrace.NewTraces()
	rs := traces.ResourceSpans().AppendEmpty()
	ss := rs.ScopeSpans().AppendEmpty()
	span := ss.Spans().AppendEmpty()
	span.SetName("test-span")

	t.Log("Attempting to send traces through ExporterHelper with WaitForResult: true...")

	// This is where the deadlock occurs!
	// When WaitForResult: true, the ConsumeTraces() call blocks waiting for the result,
	// but the internal queue/batch mechanics create a deadlock
	err = exp.ConsumeTraces(ctx, traces)

	if err != nil {
		t.Errorf("Failed to consume traces through ExporterHelper: %v", err)
	} else {
		t.Log("Successfully consumed traces through ExporterHelper")
	}

	// Shutdown
	err = exp.Shutdown(ctx)
	if err != nil {
		t.Errorf("Failed to shutdown ExporterHelper: %v", err)
	}

	t.Log("Test completed - no deadlock occurred")
}
