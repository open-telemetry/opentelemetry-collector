include ../../Makefile.Common

WEAVER_VERSION=v0.16.1
WEAVER_JSONSCHEMA_SRC_DIR=tmp/weaver

.PHONY: gen-semconv
gen-semconv: $(GOJSONSCHEMA)
	mkdir -p ${WEAVER_JSONSCHEMA_SRC_DIR}
	curl -sSL https://raw.githubusercontent.com/open-telemetry/weaver/refs/tags/${WEAVER_VERSION}/schemas/semconv.schema.json > ${WEAVER_JSONSCHEMA_SRC_DIR}/schema.json
	$(GOJSONSCHEMA) \
		--struct-name-from-title \
		--package semconv \
		--output internal/semconv/generated_config.go \
		${WEAVER_JSONSCHEMA_SRC_DIR}/schema.json
