# Stabilizing environment variable resolution

## Overview

The OpenTelemetry Collector supports three different syntaxes for
environment variable resolution which differ in their syntax, semantics
and allowed variable names. Before we stabilize confmap, we need to
address several issues related to environment variables. This document
describes:

- the current (as of v0.97.0) behavior of the Collector
- the goals that an environment variable resolution should aim for
- existing deviations from these goals
- the desired behavior after making some changes

### Out of scope

CLI environment variable resolution has a single syntax (`--config env:ENV`) 
and it is considered out of scope for this document, focusing
instead on expansion within the Collector configuration.

How to get from the current to desired behavior is also considered out
of scope and will be discussed on individual PRs. It will likely involve
one or multiple feature gates, warnings and transition periods.

## Goals of an expansion system

The following are considered goals of the expansion system:

1.  ***Expansion should happen only when the user expects it***. We
    should aim to expand when the user expects it and keep the original
    value when we don't (e.g. because the syntax is used for something
    different).
2.  ***Expansion must have predictable behavior***.
3.  ***Multiple expansion methods, if present, should have similar behavior.***
    Switching from `${env:ENV}` to `${ENV}` or vice versa
    should not lead to any surprises.
4.  ***When the syntax overlaps, expansion should be aligned with*** 
    [***the expansion defined by the Configuration Working Group***](https://github.com/open-telemetry/opentelemetry-specification/blob/032213cedde54a2171dfbd234a371501a3537919/specification/configuration/file-configuration.md#environment-variable-substitution). See [opentelemetry-specification/issues/3963](https://github.com/open-telemetry/opentelemetry-specification/issues/3963) for the counterpart to this line of work in the SDK File spec.

## Current behavior

The Collector supports three different syntaxes for environment variable
resolution:

1.  The *naked syntax*, `$ENV`.
2.  The *braces syntax*, `${ENV}`.
3.  The *env provider syntax*, `${env:ENV}`.

These differ in the character set allowed for environment variable names
as well as the type of parsing they return. Escaping is supported in all
syntaxes by using two dollar signs.

### Type casting rules

A provider or converter takes a string and returns some sort of value
after potentially doing some parsing. This gets stored in a
`confmap.Conf`. When unmarshalling, we use [mapstructure](https://github.com/mitchellh/mapstructure) with
`WeaklyTypedInput` enabled, which does a lot of implicit casting. The
details of this type casting are complex and are outlined on issue
[#9532](https://github.com/open-telemetry/opentelemetry-collector/issues/9532).

When using this notation in inline mode (e.g.
`http://endpoint/${env:PATH}`) we also do manual implicit type
casting with a similar approach to mapstructure. These are outlined
[here](https://github.com/open-telemetry/opentelemetry-collector/blob/fc4c13d3c2822bec39fa9d9658836d1a020c6844/confmap/expand.go#L124-L139).

### Naked syntax

The naked syntax is supported via the expand converter. It is
implemented using the [`os.Expand`](https://pkg.go.dev/os#Expand) stdlib
function. This syntax supports identifiers made up of:

1. ASCII alphanumerics and the `_` character
2. Certain special characters if they appear alone typically used in
   Bash: `*`, `#`, `$`, `@`, `!`, `?` and `-`.

You can see supported identifiers in this example:
[`go.dev/play/p/YfxLtYbsL6j`](https://go.dev/play/p/YfxLtYbsL6j).

The environment variable value is taken as-is and the type is always
string.

### Braces syntax

The braces syntax is supported via the expand converter. It is also
implemented using the os.Expand stdlib function. This syntax supports
any identifiers that don't contain `}`. Again, refer to the os.Expand
example to see how it works in practice:
[`go.dev/play/p/YfxLtYbsL6j`](https://go.dev/play/p/YfxLtYbsL6j).

The environment variable value is taken as-is and the type is always
string.

### `env` provider

The `env` provider syntax is supported via the `env`
provider. It is a custom implementation with a syntax that supports any
identifier that does not contain a `$`. This is done to support recursive
resolution (e.g. `${env:${http://example.com}}` would get the
environment variable whose name is stored in the URL
`http://example.com`).

The environment variable value is parsed by the yaml.v3 parser to an
any-typed variable. The yaml.v3 parser mostly follows the YAML v1.2
specification with [*some exceptions*](https://github.com/go-yaml/yaml#compatibility). 
You can see how it works for some edge cases in
[this go.dev/play example](https://go.dev/play/p/3vNLznwSZQe).

### Issues of current behavior

#### Unintuitive behavior on unset environment variables

When an environment variable is empty, all syntaxes return an empty
string with no warning given; this is frequently unexpected but can also
be used intentionally. This is especially unintuitive when the user did
not expect expansion to happen. Three examples where this is unexpected
are the following:

1.  **Opaque values such as passwords that contain `$`** (issue
    [#8215](https://github.com/open-telemetry/opentelemetry-collector/issues/8215)).
    If the $ is followed by an alphanumeric character or one of the
    special characters, it's going to lead to false positives.
2.  **Prometheus relabel config** (issue
    [`contrib#9984`](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/9984)).
    Prometheus uses `${1}` in some of its configuration values. We
    resolve this to the value of the environment variable with name
    '`1`'.
3.  **Other uses of $** (issue
    [`contrib#11846`](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/11846)).
    If a product requires the use of `$` in some field, we would most
    likely interpret it as an environment variable. This is not
    intuitive for users.

#### Unexpected type casting

When using the env syntax we parse its value as YAML. Even if you are
familiar with YAML, because of the implicit type casting rules and the
way we store intermediate values, we can get unintuitive results.

The most clear example of this is issue
[*#8565*](https://github.com/open-telemetry/opentelemetry-collector/issues/8565):
When setting a variable to value `0123` and using it in a string-typed
field, it will end up as the string `"83"` (where as the user would
expect the string to be `0123`).

#### We are less restrictive than the Configuration WG

The Configuration WG defines an [*environment variable expansion feature
for SDK
configurations*](https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/configuration/data-model.md#environment-variable-substitution).
This accepts only non empty alphanumeric + underscore identifiers
starting with alphabetic or underscore. If the Configuration WG were to
expand this in the future (e.g. to include other features present in
Bash-like syntax as in [opentelemetry-specification/pull/3948](https://github.com/open-telemetry/opentelemetry-specification/pull/3948)), we would not be able to expand our braces syntax to
support new features without breaking users.

## Desired behavior

*This section is written as if the changes were already implemented.*

The Collector supports **two** different syntaxes for environment
variable resolution:

1.  The *braces syntax*, `${ENV}`.
2.  The *env provider syntax*, `${env:ENV}`.

These both have **the same character set and behavior**. They both use
the env provider under the hood. This means we support the exact same
syntax as the Configuration WG.

The naked syntax supported in Bash is not supported in the Collector.
Escaping is supported by using two dollar signs. Escaping is also
honored for unsupported identifiers like `${1}` (i.e. anything that 
matches `\${[^$}]+}`).

### Type casting rules

The environment variable value is parsed by the yaml.v3 parser to an
any-typed variable and the original representation as a string is also stored. 
The `yaml.v3` parser mostly follows the YAML v1.2 specification with [*some
exceptions*](https://github.com/go-yaml/yaml#compatibility). You can see
how it works for some edge cases in this example:
[*https://go.dev/play/p/RtPmH8aZA1X*](https://go.dev/play/p/RtPmH8aZA1X).

When unmarshalling, we use mapstructure with WeaklyTypedInput
**disabled**. We check via a hook the original string representation of the data
and use its return value when it is valid and we are mapping to a string
field. This method has default casting rules for unambiguous scalar
types but may return the original representation depending on the
construction of confmap.Conf (see the comparison table below for details).

For using this notation in inline mode (e.g.`http://endpoint/${env:PATH}`), we
use the original string representation as well (see the comparison table below for details).

### Character set

An environment variable identifier must be a nonempty ASCII alphanumeric
or underscore starting with an alphabetic or underscore character. Its
maximum length is 200 characters. Both syntaxes support recursive
resolution.

When an invalid identifier is found, an error is emitted. To use an invalid
identifier, the string must be escaped.

### Comparison table with current behavior

This is a comparison between the current and desired behavior for
loading a field with the braces syntax, `env` syntax.

| Raw value    | Field type | Current behavior, `${ENV}`, single field | Current behavior,  `${env:ENV}` , single field | Desired behavior, entire field | Desired behavior, inline string field |
|--------------|------------|------------------------------------------|------------------------------------------------|--------------------------------|---------------------------------------|
| `123`        | integer    | 123                                      | 123                                            | 123                            | n/a                                   |
| `0123`       | integer    | 83                                       | 83                                             | 83                             | n/a                                   |
| `0123`       | string     | 0123                                     | 83                                             | 0123                           | 0123                                  |
| `0xdeadbeef` | string     | 0xdeadbeef                               | 3735928559                                     | 0xdeadbeef                     | 0xdeadbeef                            |
| `"0123"`     | string     | "0123"                                   | 0123                                           | "0123"                         | "0123"                                |
| `!!str 0123` | string     | !!str 0123                               | 0123                                           | !!str 0123                     | !!str 0123                            |
| `t`          | boolean    | true                                     | true                                           | Error: mapping string to bool  | n/a                                   |
| `23`         | boolean    | true                                     | true                                           | Error: mapping integer to bool | n/a                                   |
