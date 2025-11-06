# Code Coverage Requirements Guide for Component Authors

## Overview

As of this implementation, all OpenTelemetry Collector components are subject to automated code coverage validation based on their stability level.

## Default Requirements

### Stable Components

- **Required**: 80% minimum code coverage
- **Automatic**: Enforced by CI for all components with `stable` in their stability map

### Other Stability Levels (Alpha, Beta, Development)

- **Required**: No default minimum
- **Optional**: Can set custom requirements (see below)

## Setting Custom Coverage Requirements

You can specify a higher coverage requirement for your component by adding the `coverage_minimum` field to your `metadata.yaml`:

```yaml
type: mycomponent
status:
  class: receiver
  stability:
    stable: [metrics, traces]
  coverage_minimum: 85 # Require 85% coverage instead of the default 80%
```

### Important Notes

1. **Higher Standards Only**: The `coverage_minimum` field can only increase the requirement, not decrease it. Setting it to a value lower than the stability-based minimum (80% for stable) has no effect.

2. **Value Range**: Must be between 0 and 100. Values outside this range will cause validation errors.

3. **Repository Minimum**: If a repository-wide minimum is set (via CI configuration), your component must meet whichever is higher: the stability-based minimum, the repository minimum, or your custom `coverage_minimum`.

## Examples

### Example 1: Stable Component with Default Coverage

```yaml
type: otlp
status:
  class: receiver
  stability:
    stable: [metrics, traces, logs]
  # No coverage_minimum specified
```

**Result**: Must have ≥80% coverage

### Example 2: Stable Component with Higher Requirement

```yaml
type: reliable
status:
  class: exporter
  stability:
    stable: [metrics]
  coverage_minimum: 90
```

**Result**: Must have ≥90% coverage

### Example 3: Alpha Component with Coverage Goal

```yaml
type: experimental
status:
  class: processor
  stability:
    alpha: [metrics]
  coverage_minimum: 70
```

**Result**: Must have ≥70% coverage (helps ensure quality even at alpha stage)

### Example 4: Beta Component Preparing for Stable

```yaml
type: maturing
status:
  class: connector
  stability:
    beta: [traces]
  coverage_minimum: 80 # Preparing for stable promotion
```

**Result**: Must have ≥80% coverage (matching stable requirements)

## Checking Your Component's Coverage

### Locally

You can check your component's coverage requirements and actual coverage:

```bash
# Generate coverage data
make gotest-with-cover

# Run the coverage validator
cd cmd/checkcover
go run . -c ../../coverage.txt -r ../.. -v
```

### In CI

Coverage validation runs automatically in CI after tests complete. If your component doesn't meet requirements, the build will fail with a clear error message:

```
❌ receiver/mycomponent: coverage 75.00% is below minimum 80% (stability: stable, module: go.opentelemetry.io/collector/receiver/mycomponent)
```

## Disabling Coverage Badge

If you want to disable the Codecov badge in your component's README (not recommended), you can use:

```yaml
status:
  disable_codecov_badge: true
```

**Note**: This only hides the badge; coverage validation still runs.

## Best Practices

1. **Set Realistic Goals**: If your component is stable, it should already meet or exceed 80% coverage. If not, work on adding tests before promoting to stable.

2. **Incremental Improvement**: For alpha/beta components, consider setting a `coverage_minimum` that represents your goal, even if it's not required. This helps track progress.

3. **Document Uncovered Code**: If certain code paths are difficult to test, document why in comments and consider if the design could be improved.

4. **Review Coverage Reports**: Don't just aim for a percentage - ensure your tests are meaningful and cover important code paths.

5. **CI First**: Always check that your changes pass coverage validation in CI before merging.

## Troubleshooting

### "Coverage X% is below minimum Y%"

- **Solution**: Add more tests to your component to increase coverage
- **Alternative**: If your component has difficult-to-test code, consider refactoring for testability

### "Could not determine module path"

- **Cause**: The tool couldn't find a `go.mod` file for your component
- **Solution**: Ensure your component is in a proper Go module structure

### "No coverage data found for module"

- **Cause**: No test coverage was collected for your component
- **Solution**: Ensure you have tests and they're being run by `make gotest-with-cover`

## Getting Help

If you encounter issues with coverage validation:

1. Check the [Component Stability](../../docs/component-stability.md) document
2. Review the [checkcover README](../../cmd/checkcover/README.md)
3. Ask in the #otel-collector Slack channel
4. Open an issue on GitHub with the label `coverage`

## Future Considerations

The coverage requirements and validation tooling may evolve. Potential future changes include:

- Default minimums for beta (e.g., 60%) and alpha (e.g., 40%) components
- Coverage trend tracking
- Exemptions for specific files or packages
- HTML coverage reports

Stay tuned to repository announcements for any changes to coverage requirements.
