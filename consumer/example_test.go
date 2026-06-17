// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumer_test

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
)

func Example() {
	// Create a Logs consumer using a ConsumeLogsFunc.
	// By default, a consumer reports that it does not mutate data.
	logs, err := consumer.NewLogs(func(_ context.Context, ld plog.Logs) error {
		fmt.Printf("received %d log record(s)\n", ld.LogRecordCount())
		return nil
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("mutates data: %v\n", logs.Capabilities().MutatesData)

	// Send some log data through the consumer.
	ld := plog.NewLogs()
	ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()

	if err := logs.ConsumeLogs(context.Background(), ld); err != nil {
		panic(err)
	}

	// Output:
	// mutates data: false
	// received 1 log record(s)
}

func Example_withCapabilities() {
	// Create a Logs consumer that declares it mutates the data it receives.
	// Processors that modify data in place should set MutatesData to true so
	// that the pipeline knows to copy data before passing it to other consumers.
	logs, err := consumer.NewLogs(
		func(_ context.Context, ld plog.Logs) error {
			fmt.Printf("processed %d log record(s)\n", ld.LogRecordCount())
			return nil
		},
		consumer.WithCapabilities(consumer.Capabilities{MutatesData: true}),
	)
	if err != nil {
		panic(err)
	}

	fmt.Printf("mutates data: %v\n", logs.Capabilities().MutatesData)

	// Output:
	// mutates data: true
}
