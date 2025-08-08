# OpenTelemetry Wildcard Name Matching

## Overview

### What and Why

The following is a specification for OpenTelemetry Wildcard Name Matching grammar that could be added to the generated `metadata` packages from `mdatagen`. This grammar will facilitate the use of wildcards in metric and resource attribute configurations to enabling/disabling metrics/resource attributes in a more concise way, allowing a configuration like:

```yaml
metrics:
  "*":
    enabled: false
  system.*:
    enabled: true
```

Previously, this would require every metric be listed individually to configure. With this feature, we can allow this to be specified in two simple entries.

### Challenges

The biggest challenge with this feature is that the order specified in the configuration cannot be relied on. Since the underlying type is a Go `map`, order is not deterministic. This is the main reason that I propose coming up with a custom grammar rather than using an existing solution; this specification will include the prioritization system that determines the order of patterns being processed, regardless of the order received.

The grammar being custom means that we will need to manually implement a matching engine. I will include the specification for that as well.

## Specification

### Grammar

The grammar is an extension of the OTel name grammar. In plain english, the OTel name gramma is an OTel identifier (characters with underscores) optionally followed by a `.` and another OTel identifier. With the wildcard matching, the grammar is similar, except instead of a name you can optionally have one of two things:

* A wildcard `*` character, which matches anything in a group. If it is the final group, it will match any number of subsequent groups.
* A multimatch `{x,y,z,...}` which will match anything specified within it

The two of these will be referred to as "matchers" for the remainder of the document.

Formal grammar specification can be found [at the bottom of this document](#grammar-spec).

### Priority

The priority system is required due to the fact that the part of the collector that processes these patterns receives them in a non-deterministic order. It would be ideal if they could be applied in the order specified in the configuration file, but unfortunately that is not possible.

The priority system uses the following rules in this order:

* Patterns are sorted in ascending order of number namespace groups (identifiers/wildcards separated by `.`)
* Matchers in earlier groups are applied first
* Matchers are applied before identifiers in the same group
* Wildcard matchers are applied before multimatch matchers
* If all other priority conditions don't trigger, they are applied in lexicographical order

When a priority rule is triggered, it takes precedence over any below it, i.e. if one pattern has less groups than another, it will be applied first even if the latter has an earlier matcher. This is to ensure that the order of patterns is always deterministic.

Given the following unsorted list of patterns:
```
process.cpu.{time,utilization}
process.cpu.*
processes.*
*
process.cpu.count
```
With priority applied, the patterns would be applied in the following order:
```
*
processes.*
process.cpu.*
process.cpu.{time,utilization}
process.cpu.count
```

### Examples

The following examples demonstrate using the name matching patterns with the [Semantic Conventions for HTTP Metrics](https://github.com/open-telemetry/semantic-conventions/blob/5c3e4f1d4a21bcd6482edfe5feca74b5018f8cad/docs/http/http-metrics.md) (permalinked to revision as of writing of this document, metrics may have changed if reading in the future).

---

```yaml
metrics:
  http.client.*:
    enabled: true
  "*":
    enabled: false
```

First, all metrics will be first be disabled since `*` is a higher priority pattern and will be applied first. Then, all `http.client` namespaced metrics will be enabled.

---

```yaml
metrics:
  http.*.{request,response}.body.size:
    enabled: true
```

The following metrics will be enabled:
`http.client.request.body.size`
`http.client.response.body.size`
`http.server.request.body.size`
`http.server.response.body.size`

---

```yaml
metrics:
  http.client.*:
    enabled: false
  http.*.active_requests:
    enabled: true
```

The first pattern applied would be `http.*.active_requests`, so first `http.client.active_requests` and `http.server.active_requests` would be enabled. Then, the `http.client.*` pattern would be applied, which means all `http.client` namespaced metrics would be disabled, including `http.client.active_requests`. Oops, that's not what we wanted! Unfortunately, to make this work we will need the following:
```yaml
metrics:
  http.client.*:
    enabled: false
  http.client.active_requests:
    enabled: true
  http.server.active_requests:
    enabled: true
```
The wildcard pattern `http.client.*` will be applied first, then the `active_requests` enablers will run.

### Grammar Spec

You can try the grammar out at [this BNF Playground link](https://bnfplayground.pauliankline.com/?bnf=%3Cpattern%3E%20%3A%3A%3D%20%3Cgroup%3E%20(%22.%22%20%3Cgroup%3E)*%0A%3Cgroup%3E%20%3A%3A%3D%20%3Cotel_identifier%3E%20%7C%20%3Cmatcher%3E%0A%3Cotel_identifier%3E%20%3A%3A%3D%20%3Cotel_identifier_character%3E%20(%22_%22%3F%20%3Cotel_identifier_character%3E)*%0A%3Cotel_identifier_character%3E%20%3A%3A%3D%20%3Cletter%3E%20%7C%20%3Cdigit%3E%0A%3Cmatcher%3E%20%3A%3A%3D%20%22*%22%20%7C%20%3Cmultimatch%3E%0A%3Cmultimatch%3E%20%3A%3A%3D%20%22%7B%22%20%3Cotel_identifier%3E%20(%22%2C%22%20%22%20%22%3F%20%3Cotel_identifier%3E)*%20%22%7D%22%0A%0A%3Cletter%3E%20%3A%3A%3D%20%5Ba-z%5D%0A%3Cdigit%3E%20%3A%3A%3D%20%5B0-9%5D&name=OpenTelemetry%20Name%20Matching%20Pattern).

#### EBNF

```bnf
<pattern> ::= <group> ("." <group>)*
<group> ::= <otel_identifier> | <matcher>
<otel_identifier> ::= <otel_identifier_character> ("_"? <otel_identifier_character>)*
<otel_identifier_character> ::= <letter> | <digit>
<matcher> ::= "*" | <multimatch>
<multimatch> ::= "{" <otel_identifier> ("," " "? <otel_identifier>)* "}"

<letter> ::= [a-z]
<digit> ::= [0-9]
```

#### Nearley

Try it out by pasting it into the [Nearley Playground](https://omrelli.ug/nearley-playground/).

```ne
pattern -> group ( "." group ):*
group -> otel_identifier | matcher
otel_identifier -> otel_identifier_character ( "_":? otel_identifier_character ):*
otel_identifier_character -> letter | digit
matcher -> "*" | multimatch
multimatch -> "{" otel_identifier ( "," " ":? otel_identifier ):* "}"

letter -> [a-z]
digit -> [0-9]
```
