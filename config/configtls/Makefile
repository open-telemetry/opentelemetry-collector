include ../../Makefile.Common

# The requirefips build tag changes the accepted curve_preferences values.
# Run the tests with it too, so that curves_fips.go stays covered.
.PHONY: test-requirefips
test-requirefips:
	$(GO_TOOL) gotestsum --packages="./..." -- $(GOTEST_OPT) -tags requirefips

test: test-requirefips
