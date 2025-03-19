// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package receiver defines components that allow the Collector to receive metrics, traces and logs.
//
// A receiver receives data from a source (either from a remote source via network
// or scrapes from a local host) and pushes the data to the pipelines it is attached
// to by calling the nextConsumer.Consume*() function.
//
// # Error Handling
//
// The nextConsumer.Consume*() function may return an error to indicate that the data was not
// accepted. This error should be handled as documented in the consumererror package.
//
// Depending on the error type, the receiver must indicate to the source from which it received the
// data the type of error in a protocol-dependent way, if that is supported by the receiving protocol.
// For example, a receiver for the OTLP/HTTP protocol would use the HTTP status codes as defined in
// the OTLP specification.
//
// # Acknowledgment and Checkpointing
//
// The receivers that receive data via a network protocol that support acknowledgments
// MUST follow this order of operations:
//   - Receive data from some sender (typically from a network).
//   - Push received data to the pipeline by calling nextConsumer.Consume*() function.
//   - Acknowledge successful data receipt to the sender if Consume*() succeeded or
//     return a failure to the sender if Consume*() returned an error.
//
// This ensures there are strong delivery guarantees once the data is acknowledged
// by the Collector.
//
// Similarly, receivers that use checkpointing to remember the position of last processed
// data (e.g. via storage extension) MUST store the checkpoint only AFTER the Consume*()
// call returns.
package receiver // import "go.opentelemetry.io/collector/receiver"
