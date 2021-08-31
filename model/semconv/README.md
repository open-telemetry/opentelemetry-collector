# Semantic Convention Constants

This package contains constants representing the OpenTelemetry semantic conventions.  They are automatically generated
from definitions in the specification.

## Generation

To generate the constants you can use the `gensemconv` make target.  You must provide the path to the root of a clone
of the `opentelemetry-specification` repository in the `SPECPATH` variable and the version of the conventions to
generate in the `SPECTAG` variable.

```console
$ make gensemconv SPECPATH=~/dev/opentelemetry-specification SPECTAG=v1.5.0
Generating semantic convention constants from specification version v1.5.0 at ~/dev/opentelemetry-specification
semconvgen -o model/semconv/v1.5.0 -t model/internal/semconv/template.j2 -s v1.5.0 -i ~/dev/opentelemetry-specification/semantic_conventions/resource -p conventionType=resource
semconvgen -o model/semconv/v1.5.0 -t model/internal/semconv/template.j2 -s v1.5.0 -i ~/dev/opentelemetry-specification/semantic_conventions/trace -p conventionType=trace
```

When generating the constants for a new version ot the specification it is important to note that only `trace.go` and
`resource.go` are generated automatically.  The `schema.go` and `nonstandard.go` files should be copied from a prior
version's package and updated as appropriate.  Most important will be to update the `SchemaURL` constant in `schema.go`.
