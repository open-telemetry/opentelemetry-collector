# Logging Policy for Sensitive Data

To ensure the security and privacy of data processed by the OpenTelemetry Collector, the following rules regarding logging must be observed.

## Rules

1. **No Sensitive Data at Info/Warn/Error Levels**
   - Telemetry data (metric names, label values, attribute values, resource attributes) must NOT be logged at `Info`, `Warn`, or `Error` levels.
   - PII (Personally Identifiable Information), credentials, and exact payload contents are strictly prohibited in high-level logs.

2. **Error Messages**
   - Returned errors must be generic and actionable without embedding specific metric data.
   - Do NOT wrap errors with formatted strings containing raw telemetry values (e.g., `fmt.Errorf("failed processing metric %s: %w", metricName, err)` is prohibited).
   - Use structural logging fields at `Debug` level if context is needed.

3. **Debug Level Exception**
   - Detailed diagnostic context, including specific metric names or attribute values, MAY be logged at `Debug` level only.
   - This allows operators to opt-in to verbose logging for troubleshooting without polluting default logs with potentially sensitive data.

## Implementation Guidelines

**Incorrect:**
```go
// BAD: Leaks sensitive metric name in error
if err := process(metric); err != nil {
    return fmt.Errorf("failed to process metric %s: %w", metric.Name(), err)
}
```

**Correct:**
```go
// GOOD: Logs details at debug, returns generic error
if err := process(metric); err != nil {
    logger.Debug("failed to process metric",
        zap.String("metric_name", metric.Name()),
        zap.Error(err))
    return fmt.Errorf("failed to process metric: %w", err)
}
```
