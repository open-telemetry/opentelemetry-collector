# Overview

This package implements a number of translators that help translate spans to and from OpenCensus format to a number of other supported formats such as Jaeger Proto, Jaeger Thrift, Zipkin Thrift, Zipkin JSON. This document mentions how certain non-obvious things should be handled.

## Links:

* [OpenTracing Semantic Conventions](https://github.com/opentracing/specification/blob/master/semantic_conventions.md)

## Status Codes and Messages

### OpenCensus

OpenCensus protocol has a special field to represent the status of an operation. The status field has two fields, an int32 field called `code` and a string field called `message`. When converting from other formats, status field must be set from the relevant tags/attributes of the source format. When converting from OC to other formats, the status field must be translated to appropriate tags/attributes of the target format. 


### Jaeger to OC

Jaeger spans may contain two possible sets of tags that can possibly represent the status of an operation:

- `status.code` and `status.message`
- `http.status_code` and `http.status_message`

When converting from Jaeger to OC, 

1. OC status should be set from `status.code` and `status.message` tags if `status.code` tag is found on the Jaeger span. Since OC already has a special status field, these tags (`status.code` and `status.message`) are redundant and should be dropped from resultant OC span.
2. If the `status.code` tag is not present, status should be set from `http.status_code` and `http.status_message` if the `http.status_code` tag is found. HTTP status code should be mapped to the appropriate gRPC status code before using it in OC status. These tags should be preserved and added to the resultant OC span as attributes.
3. If none of the tags are found, OC status should not be set.


### Zipkin to OC

In addition to the two sets of tags mentioned in the previous section, Zipkin spans can possibly contain a third set of tags to represent operation status resulting in the following sets of tags:

- `census.status_code` and `census.status_description`
- `status.code` and `status.message`
- `http.status_code` and `http.status_message`

When converting from Zipkin to OC,

1. OC status should be set from `census.status_code` and `census.status_description` if `census.status_code` tag is found on the Zipkin span. These tags should be dropped from the resultant OC span.
2. If the `census.status_code` tag is not found in step 1, OC status should be set from `status.code` and `status.message` tags if the `status.code` tag is present. The tags should be dropped from the resultant OC span.
3. If no tags are found in step 1 and 2, OC status should be set from `http.status_code` and `http.status_message` if either `http.status_code` tag is found. These tags should be preserved and added to the resultant OC span as attributes.
4. If none of the tags are found, OC status should not be set.


Note that codes and messages from different sets of tags should not be mixed to form the status field. For example, OC status should not contain code from `http.status_code` but message from `status.message` and vice-versa. Both fields must be set from the same set of tags even if it means leaving one of the two fields empty.


### OC to Jaeger

When converting from OC to Jaeger, if the OC span has a status field set, then

* `code` should be added as a `status.code` tag.
* `message` should be added as a `status.message` tag.

### OC to Zipkin

When converting from OC to Zipkin, if the OC span has the status field set, then

* `code` should be added as a `census.status_code` tag.
* `message` should be added as a `census.status_description` tag.

In addition to this, if the OC status field represents a non-OK status, then a tag with the key `error` should be added and set to the same value as that of the status message falling back to status code when status message is not available.

### Note:

If either target tags (`status.*` or `census.status_*`) are already present on the span, then they should be preserved and not overwritten from the status field. This is extremely unlikely to happen within the collector because of how things are implemented but any other implementations should still follow this rule.


## Converting HTTP status codes to OC codes

The following guidelines should be followed for translating HTTP status codes to OC ones. https://github.com/googleapis/googleapis/blob/master/google/rpc/code.proto

This is implemented by the `tracetranslator` package as `HTTPToOCCodeMapper`.
