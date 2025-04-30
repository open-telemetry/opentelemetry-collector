// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package receiver_test

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/receiver"
)

var typeStr = component.MustNewType("example")

type exampleConfig struct {
	Interval time.Duration
}

// Reciver must implement the Start() and Shutdown() methods so the receiver type can be compliant
// with the receiver.Traces interface.
type exampleReceiver struct {
	host         component.Host
	cancel       context.CancelFunc
	nextConsumer consumer.Traces
	config       exampleConfig
}

func (rcvr *exampleReceiver) Start(ctx context.Context, host component.Host) error {
	rcvr.host = host
	ctx, rcvr.cancel = context.WithCancel(ctx)

	go func() {
		ticker := time.NewTicker(rcvr.config.Interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Your receiver processing code should come here
				err := rcvr.nextConsumer.ConsumeTraces(ctx, generateTrace())
				if err != nil {
					fmt.Println("Error when consuming trace: %w", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

func (rcvr *exampleReceiver) Shutdown(_ context.Context) error {
	if rcvr.cancel != nil {
		rcvr.cancel()
	}
	return nil
}

func generateTrace() ptrace.Traces {
	traces := ptrace.NewTraces()

	// In reallity you may need to fetch or receive and transform
	// some telemetry data.
	// For this example we will just generate some dummy data
	resourceSpan := traces.ResourceSpans().AppendEmpty()
	resource := resourceSpan.Resource()

	attrs := resource.Attributes()
	attrs.PutInt("id", 1)

	scopeSpans := resourceSpan.ScopeSpans().AppendEmpty()
	scopeSpans.Scope().SetName("example-system")
	scopeSpans.Scope().SetVersion("v1.0")

	traceID := pcommon.TraceID([]byte("1"))
	spanID := pcommon.SpanID([]byte("2"))

	span := scopeSpans.Spans().AppendEmpty()
	span.SetTraceID(traceID)
	span.SetSpanID(spanID)
	span.SetName("Operation 1")
	span.SetKind(ptrace.SpanKindClient)
	span.Status().SetCode(ptrace.StatusCodeOk)
	span.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	span.SetEndTimestamp(pcommon.NewTimestampFromTime(time.Now().Add(4 * time.Millisecond)))

	// Other resources and spans could also be added.

	return traces
}

func createDefaultConfig() component.Config {
	return &exampleConfig{
		Interval: time.Minute,
	}
}

func createExampleReceiver(_ context.Context, _ receiver.Settings,
	baseCfg component.Config, consumer consumer.Traces,
) (receiver.Traces, error) {
	exampleCfg := baseCfg.(*exampleConfig)

	rcvr := &exampleReceiver{
		nextConsumer: consumer,
		config:       *exampleCfg,
	}

	return rcvr, nil
}

func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		typeStr,
		createDefaultConfig,
		receiver.WithTraces(createExampleReceiver, component.StabilityLevelAlpha))
}

func Example() {
	// The NewFactory method can then be used on the Collector's initialization process
	// on your components.go file.

	// In this example we will just instantiate and print it's type

	exampleReceiver := NewFactory()
	fmt.Println(exampleReceiver.Type())

	// Output:
	// example
}
