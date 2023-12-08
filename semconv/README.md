# Semantic Convention Constants

This package contains constants representing the OpenTelemetry semantic conventions. They are automatically generated
from definitions in the specification.

## Generation

To generate the constants you can use the `gensemconv` make target. You must provide the path to the root of a clone of
the `semantic-conventions` repository in the `SPECPATH` variable and the version of the conventions to generate
in the `SPECTAG` variable.

```console
$ make gensemconv SPECPATH=/tmp/semantic-conventions SPECTAG=v1.22.0
Generating semantic convention constants from specification version v1.22.0 at /tmp/semantic-conventions
.tools/semconvgen -o semconv/v1.22.0 -t semconv/template.j2 -s v1.22.0 -i /tmp/semantic-conventions/model/. --only=resource -p conventionType=resource -f generated_resource.go
.tools/semconvgen -o semconv/v1.22.0 -t semconv/template.j2 -s v1.22.0 -i /tmp/semantic-conventions/model/. --only=event -p conventionType=event -f generated_event.go
.tools/semconvgen -o semconv/v1.22.0 -t semconv/template.j2 -s v1.22.0 -i /tmp/semantic-conventions/model/. --only=span -p conventionType=trace -f generated_trace.go
```

When generating the constants for a new version ot the specification it is important to note that only
`generated_trace.go` and `generated_resource.go` are generated automatically. The `schema.go` and `nonstandard.go`
files should be copied from a prior version's package and updated as appropriate. Most important will be to update
the `SchemaURL` constant in `schema.go`.
