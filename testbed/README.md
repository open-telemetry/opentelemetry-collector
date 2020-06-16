# OpenTelemetry Collector Testbed

Testbed is a controlled environment and tools for conducting end-to-end tests for the Otel Collector,
including reproducible short-term benchmarks, correctness tests, long-running stability tests and 
maximum load stress tests.

## Usage

For each type of tests that should have a summary report create a new directory and then a test suite function
which utilizes `*testing.M`. This function should delegate all functionality to `testbed.DoTestMain` supplying
a global instance of `testbed.TestResultsSummary` to it.

Each test case within the suite should create a `testbed.TestCase` and supply implementations of each of the various
interfaces the `NewTestCase` function takes as parameters.

## Pluggable Test Components

* `DataProvider` - Generates test data to send to receiver under test.
  * `PerfTestDataProvider` - Implementation of the `DataProvider` for use in performance tests. Tracing IDs are based on the incremented batch and data items counters.
  * `GoldenDataProvider` - Implementation of `DataProvider` for use in correctness tests. Provides data from the "Golden" dataset generated using pairwise combinatorial testing techniques.
* `DataSender` - Sends data to the collector instance under test.
  * `JaegerGRPCDataSender` - Implementation of `DataSender` which sends to `jaeger` receiver.
  * `OCTraceDataSender` - Implementation of `DataSender` which sends to `opencensus` receiver.
  * `OCMetricsDataSender` - Implementation of `DataSender` which sends to `opencensus` receiver.
  * `OTLPTraceDataSender` - Implementation of `DataSender` which sends to `otlp` receiver.
  * `OTLPMetricsDataSender` - Implementation of `DataSender` which sends to `otlp` receiver.
  * `ZipkinDataSender` - Implementation of `DataSender` which sends to `zipkin` receiver.
* `DataReceiver` - Receives data from the collector instance under test and stores it for use in test assertions.
  * `OCDataReceiver` - Implementation of `DataReceiver` which receives data from `opencensus` exporter.
  * `JaegerDataReceiver` - Implementation of `DataReceiver` which receives data from `jaeger` exporter.
  * `OTLPDataReceiver` - Implementation of `DataReceiver` which receives data from `otlp` exporter.
  * `ZipkinDataReceiver` - Implementation of `DataReceiver` which receives data from `zipkin` exporter.
* `OtelcolRunner` - Configures, starts and stops one or more instances of otelcol which will be the subject of testing being executed.
  * `ChildProcess` - Implementation of `OtelcolRunner` runs a single otelcol as a child process on the same machine as the test executor.
  * `InProcessCollector` - Implementation of `OtelcolRunner` runs a single otelcol as a go routine within the same process as the test executor.
* `TestCaseValidator` - Validates and reports on test results.
  * `PerfTestValidator` - Implementation of `TestCaseValidator` for test suites using `PerformanceResults` for summarizing results.
  * `CorrectnessTestValidator` - Implementation of `TestCaseValidator` for test suites using `CorrectnessResults` for summarizing results.
* `TestResultsSummary` - Records itemized test case results plus a summary of one category of testing.
  * `PerformanceResults` - Implementation of `TestResultsSummary` with fields suitable for reporting performance test results.
  * `CorrectnessResults` - Implementation of `TestResultsSummary` with fields suitable for reporting data translation correctness test results.
