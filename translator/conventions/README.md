# Semantic Convention Constants

This package contains constants representing the OpenTelemetry semantic conventions.  They are automatically generated
from definitions in the specification.

## Generation

To generate the constants you can use the `gensemconv` make target.  You must provide the path to the root of a clone
of the `opentelemetry-specification` repository in the `SPECPATH` variable.

```console
$ make gensemconv SPECPATH=~/dev/opentelemetry-specification
Generating semantic convention constants from specification at ~/dev/opentelemetry-specification
semconvgen -o translator/conventions -t translator/conventions/template.j2 -i ~/dev/opentelemetry-specification/semantic_conventions/resource -p conventionType=resource
semconvgen -o translator/conventions -t translator/conventions/template.j2 -i ~/dev/opentelemetry-specification/semantic_conventions/trace -p conventionType=trace
```
