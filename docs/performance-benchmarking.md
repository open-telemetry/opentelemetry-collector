# Performance Benchmarking

This document describes how performance benchmarking works in the OpenTelemetry Collector project, including how to write benchmarks, run them locally, and how performance regressions are detected in CI.

## Overview

The OpenTelemetry Collector uses a combination of Go's built-in benchmarking framework and [CodSpeed](https://codspeed.io) for continuous performance monitoring. This infrastructure helps us detect performance regressions early and track performance improvements over time.

## Running Benchmarks Locally

### Running All Benchmarks

To run all benchmarks across all modules:

```shell
make gobenchmark
```

This will:
- Set the `MEMBENCH` environment variable to include memory benchmarks
- Run benchmarks in all modules
- Generate a `benchmarks.txt` file with results

### Running Time-Only Benchmarks

Some benchmarks run too quickly to provide meaningful duration results. To skip them:

```shell
make timebenchmark
```

### Running Benchmarks for a Specific Module

Navigate to the module directory and run:

```shell
go test -bench=. -benchtime=1s ./...
```

### Running a Specific Benchmark

To run a specific benchmark function:

```shell
go test -bench=BenchmarkSpecificFunction -benchtime=1s ./path/to/package
```

## Writing Benchmarks

Benchmarks in the OpenTelemetry Collector follow Go's standard benchmarking practices.

### Basic Benchmark Structure

```go
func BenchmarkMyFunction(b *testing.B) {
    // Setup code here (not measured)
    data := setupTestData()

    for b.Loop() {
        // Code to benchmark
        MyFunction(data)
    }
}
```

### Memory Benchmarks

Memory benchmarks are skipped in CI unless the `MEMBENCH` environment variable is set. Use the `testutil.SkipMemoryBench` helper:

```go
func BenchmarkMemoryUsage(b *testing.B) {
    testutil.SkipMemoryBench(b)

    for b.Loop() {
        // Memory-intensive operation
    }
}
```

### Parallel Benchmarks

For benchmarks that should run in parallel:

```go
func BenchmarkParallel(b *testing.B) {
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            // Concurrent operation
        }
    })
}
```

### Sub-benchmarks

Use sub-benchmarks to test different scenarios:

```go
func BenchmarkCompression(b *testing.B) {
    benchmarks := []struct {
        name string
        codec string
    }{
        {"gzip", "gzip"},
        {"zstd", "zstd"},
        {"snappy", "snappy"},
    }

    for _, bm := range benchmarks {
        b.Run(bm.name, func(b *testing.B) {
            for b.Loop() {
                compress(data, bm.codec)
            }
        })
    }
}
```

## Continuous Performance Monitoring with CodSpeed

### How It Works

CodSpeed is integrated into our CI pipeline via the `.github/workflows/go-benchmarks.yml` workflow. It:

1. **Runs automatically** on every push to `main` and on every pull request
2. **Measures performance** using walltime instrumentation for consistent results
3. **Detects regressions** by comparing benchmark results against the base branch
4. **Posts results** as comments on pull requests showing performance changes
5. **Generates flamegraphs** for detailed performance analysis

### Interpreting CodSpeed Results

When CodSpeed detects a performance change, it will:

- **Show percentage changes** for each affected benchmark
- **Mark significant regressions** that exceed the threshold
- **Provide flamegraphs** to visualize where time is spent
- **Link to the dashboard** for historical performance trends

A typical CodSpeed comment might show:

```
⚡ Performance Impact: 2 regressions detected

- BenchmarkQueueProcessing: -15.2% slower
- BenchmarkDataParsing: -3.1% slower
```

### Viewing Performance History

The CodSpeed dashboard provides historical performance data:

- **Dashboard URL**: https://codspeed.io/open-telemetry/opentelemetry-collector
- View trends over time for all benchmarks
- Compare performance across commits and branches
- Review flamegraphs for detailed analysis

## Best Practices for Performance-Sensitive Changes

When making changes that might affect performance:

1. **Run benchmarks locally** before and after your changes
2. **Check for regressions** in the CodSpeed PR comment
3. **Document intentional trade-offs** if a regression is acceptable for correctness or features
4. **Add new benchmarks** for new performance-critical code paths
5. **Use profiling tools** (pprof) to understand bottlenecks if needed

### Using pprof for Detailed Analysis

For CPU profiling:

```shell
go test -bench=BenchmarkMyFunction -cpuprofile=cpu.prof ./...
go tool pprof cpu.prof
```

For memory profiling:

```shell
go test -bench=BenchmarkMyFunction -memprofile=mem.prof ./...
go tool pprof mem.prof
```

## Performance Regression Policy

Performance regressions are treated seriously:

- **Unintentional regressions** should be addressed before merging
- **Intentional trade-offs** must be documented and justified in the PR description
- **Large regressions** (>10%) require extra scrutiny and approval from maintainers
- **Small regressions** (<5%) may be acceptable if they improve code quality or add necessary features

## Recent Performance Changes

For a list of recent performance-related changes, see:

```shell
git log --all --oneline --grep="perf\|benchmark\|optim" --since="3 months ago"
```

## Troubleshooting

### Benchmarks Are Flaky

If benchmarks show inconsistent results:
- Ensure your machine is not under heavy load
- Increase `-benchtime` for longer, more stable measurements
- Consider using `-count=10` to run multiple iterations
- Check if background processes are interfering

### CodSpeed Shows Unexpected Regression

1. Run benchmarks locally to confirm the regression
2. Use pprof to identify the source of slowdown
3. Check if the regression is due to test data changes rather than code changes
4. Review the flamegraphs in CodSpeed for insights

## Additional Resources

- [Go Benchmarking Guide](https://pkg.go.dev/testing#hdr-Benchmarks)
- [CodSpeed Documentation](https://docs.codspeed.io)
- [Profiling Go Programs](https://go.dev/blog/pprof)
- [Internal Architecture](internal-architecture.md) for understanding collector components
