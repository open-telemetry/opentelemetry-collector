# Pull Request Summary: Validate and Enforce Test Coverage Requirements (#14098)

## Description

This PR implements automated validation and enforcement of code coverage requirements for OpenTelemetry Collector components based on their stability level, as specified in issue #14098.

## Changes

### 1. Metadata Schema Enhancement

- Added optional `coverage_minimum` field to `metadata.yaml` schema
- Allows components to specify custom minimum coverage requirements
- File: `cmd/mdatagen/metadata-schema.yaml`

### 2. Status Struct Update

- Added `CoverageMinimum` field with validation (0-100 range)
- File: `cmd/mdatagen/internal/status.go`

### 3. Coverage Validation Tool (`checkcover`)

Created a new CLI tool that:

- Parses coverage data from `coverage.txt`
- Finds all components via their `metadata.yaml` files
- Determines minimum coverage based on:
  - Stability level (80% for stable components)
  - Repository minimum (if specified)
  - Component-specific `coverage_minimum`
- Validates actual coverage meets requirements
- Provides detailed pass/fail reporting

Files:

- `internal/cmd/checkcover/main.go` - CLI entry point
- `internal/cmd/checkcover/validator.go` - Core logic
- `internal/cmd/checkcover/validator_test.go` - Tests
- `internal/cmd/checkcover/README.md` - Documentation
- `internal/cmd/checkcover/go.mod` - Dependencies
- `internal/cmd/checkcover/Makefile` - Build config

### 4. CI Integration

- Added coverage validation step to `.github/workflows/build-and-test.yml`
- Runs after coverage collection but before uploads
- Fails build if any component doesn't meet requirements

### 5. Documentation

- `docs/coverage-requirements.md` - Guide for component authors
- `COVERAGE_VALIDATION_IMPLEMENTATION.md` - Implementation details

## Coverage Requirements

### Stable Components

- **Minimum**: 80% coverage (as per Component Stability document)
- **Override**: Can set higher via `coverage_minimum` in metadata.yaml
- **Enforced**: Automatically in CI

### Other Stability Levels

- **Minimum**: None by default
- **Optional**: Components can set `coverage_minimum` for their own quality goals
- **Flexible**: Helps components prepare for stable promotion

## Usage Example

### Component with Custom Minimum

```yaml
# metadata.yaml
type: myreceiver
status:
  class: receiver
  stability:
    stable: [metrics, traces]
  coverage_minimum: 90 # Require 90% instead of default 80%
```

### Running the Tool

```bash
# Locally
make gotest-with-cover
cd internal/cmd/checkcover
go run . -c ../../../coverage.txt -r ../../.. -v

# In CI (automatic)
# Runs as part of test-coverage job
```

## Benefits

1. ✅ **Automated**: No manual checking required
2. ✅ **Clear Standards**: Components know exactly what's required
3. ✅ **Flexible**: Components can set higher standards
4. ✅ **CI Enforced**: Prevents merging undertested code
5. ✅ **Visible**: Coverage badges already shown in READMEs

## Testing

- All existing mdatagen tests pass
- New checkcover tool builds and runs successfully
- Help output and CLI interface verified

## Addresses Issue Requirements

From #14098:

✅ **Make sure code coverage is shown for all components**

- Already handled by existing codecovgen functionality
- Verified template generates badges correctly
- Uses `GetCodeCovComponentID()` for ID matching

✅ **Introduce a CI check for coverage requirements**

- Implemented as `checkcover` tool
- Integrated into CI workflow
- Parses coverage data using `go tool covdata textfmt` output
- Validates against metadata.yaml settings
- Fails build with clear error messages

✅ **Support for component-specific minimums**

- Added `coverage_minimum` field to metadata.yaml
- Validated on parsing (must be 0-100)
- Applied as highest of stability/repo/component minimums

## Future Enhancements

Potential improvements for future PRs:

- Default minimums for beta (60%) and alpha (40%) components
- Coverage trend tracking
- Exemptions for specific files
- HTML coverage reports
- Integration with coverage dashboards

## Breaking Changes

None. This is purely additive:

- New optional field in metadata.yaml
- New validation tool (doesn't affect existing workflows)
- CI check added but all existing stable components should already meet 80% requirement

## Related Issues

- Closes #14098
- Part of #14066 (Component Stability automation)

## Checklist

- [x] Code compiles and builds successfully
- [x] All existing tests pass
- [x] New functionality tested
- [x] Documentation added/updated
- [x] CI integration complete
- [x] No breaking changes
- [x] Follows coding guidelines

## Notes for Reviewers

1. **checkcover tool**: Main implementation in `internal/cmd/checkcover/`
2. **Metadata changes**: Simple addition of one field with validation
3. **CI changes**: Single step added to existing workflow
4. **Documentation**: Comprehensive guide for component authors

Please review:

- Tool logic for determining minimum coverage
- Coverage parsing accuracy
- CI integration approach
- Documentation clarity
