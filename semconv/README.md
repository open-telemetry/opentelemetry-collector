# Semantic Convention Constants

This package contains constants representing the OpenTelemetry semantic conventions. They are automatically generated
from definitions in the specification.

## Generation

To generate the constants you can use the `gensemconv` make target. You must provide the version of the conventions to generate
in the `SPECTAG` variable.

```console
$ make gensemconv SPECTAG=v1.22.0
Generating semantic convention constants from specification version v1.27.0 at https://github.com/open-telemetry/semantic-conventions.git@v1.27.0
mkdir -p /home/joshuasuereth/src/open-telemetry/opentelemetry-collector/semconv/v1.27.0
mkdir -p ~/.weaver
docker run --rm \
	-u 110055:89939 \
	--env HOME=/tmp/weaver \
	--mount 'type=bind,source=/home/joshuasuereth/src/open-telemetry/opentelemetry-collector/semconv/weaver,target=/home/weaver/templates,readonly' \
	--mount 'type=bind,source=/home/joshuasuereth/.weaver,target=/tmp/weaver/.weaver' \
	--mount 'type=bind,source=/home/joshuasuereth/src/open-telemetry/opentelemetry-collector/semconv/v1.27.0,target=/home/weaver/target' \
	otel/weaver:v0.9.1 registry generate \
	--registry=https://github.com/open-telemetry/semantic-conventions.git@v1.27.0#model \
	--templates=/home/weaver/templates \
	collector \
	/home/weaver/target
✔ `main` semconv registry `https://github.com/open-telemetry/semantic-conventions.git@v1.27.0#model[model]` loaded (160 files)
✔ No `before_resolution` policy violation
✔ `default` semconv registry resolved
✔ Generated file "/home/weaver/target/generated_resource.go"
✔ Generated file "/home/weaver/target/generated_event.go"
✔ Generated file "/home/weaver/target/generated_trace.go"
✔ Generated file "/home/weaver/target/generated_attribute_group.go"
✔ Artifacts generated successfully
```
