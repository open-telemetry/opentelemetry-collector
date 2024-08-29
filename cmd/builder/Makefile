include ../../Makefile.Common

.PHONY: ocb
ocb:
	CGO_ENABLED=0 $(GOCMD) build -trimpath -o ../../bin/ocb_$(GOOS)_$(GOARCH) .

# Generate the default build config from otelcorecol, by removing the
# "replaces" stanza, which is assumed to be at the end of the file.
#
# The default config file is checked in so that `go install` will work
# and so that non-unix builds don't need sed to be installed.
.PHONY: config
config: internal/config/default.yaml
	sed '-e/replaces:/,$$d' <../otelcorecol/builder-config.yaml > internal/config/default.yaml
