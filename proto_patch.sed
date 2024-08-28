s+go.opentelemetry.io/proto/otlp/+go.opentelemetry.io/collector/pdata/internal/data/protogen/+g

s+package opentelemetry.proto.\(.*\).v\(.*\);+package opentelemetry.proto.\1.v\2;\
\
import "gogoproto/gogo.proto";+g

s+bytes trace_id = \(.*\);+bytes trace_id = \1\
  [\
  // Use custom TraceId data type for this field.\
  (gogoproto.nullable) = false,\
  (gogoproto.customtype) = "go.opentelemetry.io/collector/pdata/internal/data.TraceID"\
  ];+g

s+bytes \(.*span_id\) = \(.*\);+bytes \1 = \2\
  [\
  // Use custom SpanId data type for this field.\
  (gogoproto.nullable) = false,\
  (gogoproto.customtype) = "go.opentelemetry.io/collector/pdata/internal/data.SpanID"\
  ];+g

s+repeated opentelemetry.proto.common.v1.KeyValue \(.*\);+repeated opentelemetry.proto.common.v1.KeyValue \1\
  [ (gogoproto.nullable) = false ];+g

s+repeated KeyValue \(.*\);+repeated KeyValue \1\
  [ (gogoproto.nullable) = false ];+g

s+AnyValue \(.*\);+AnyValue \1\
  [ (gogoproto.nullable) = false ];+g

s+opentelemetry.proto.resource.v1.Resource resource = \(.*\);+opentelemetry.proto.resource.v1.Resource resource = \1\
  [ (gogoproto.nullable) = false ];+g

s+opentelemetry.proto.common.v1.InstrumentationScope scope = \(.*\);+opentelemetry.proto.common.v1.InstrumentationScope scope = \1\
  [ (gogoproto.nullable) = false ];+g

s+Status \(.*\);+Status \1\
  [ (gogoproto.nullable) = false ];+g

s+repeated Exemplar exemplars = \(.*\);+repeated Exemplar exemplars = \1\
  [ (gogoproto.nullable) = false ];+g

s+Buckets \(.*\)tive = \(.*\);+Buckets \1tive = \2\
  [ (gogoproto.nullable) = false ];+g

# optional fixed64 foo = 1 -> oneof foo_ { fixed64 foo = 1;}
s+optional \(.*\) \(.*\) = \(.*\);+ oneof \2_ { \1 \2 = \3;}+g

s+\(.*\)PartialSuccess partial_success = \(.*\);+\1PartialSuccess partial_success = \2\
  [ (gogoproto.nullable) = false ];+g

#
# Profiles replacements
#
s+opentelemetry.proto.profiles.v1experimental.Profile \(.*\);+opentelemetry.proto.profiles.v1experimental.Profile \1\
  [ (gogoproto.nullable) = false ];+g

s+repeated ValueType \(.*\);+repeated ValueType \1\
  [ (gogoproto.nullable) = false ];+g

s+repeated Sample \(.*\);+repeated Sample \1\
  [ (gogoproto.nullable) = false ];+g

s+repeated Mapping \(.*\);+repeated Mapping \1\
  [ (gogoproto.nullable) = false ];+g

s+repeated Location \(.*\);+repeated Location \1\
  [ (gogoproto.nullable) = false ];+g

s+repeated Label \(.*\);+repeated Label \1\
  [ (gogoproto.nullable) = false ];+g

s+repeated Function \(.*\);+repeated Function \1\
  [ (gogoproto.nullable) = false ];+g

s+repeated AttributeUnit \(.*\);+repeated AttributeUnit \1\
  [ (gogoproto.nullable) = false ];+g

s+repeated Link link_table \(.*\);+repeated Link link_table \1\
  [ (gogoproto.nullable) = false ];+g

s+repeated Line \(.*\);+repeated Line \1\
  [ (gogoproto.nullable) = false ];+g

s+ValueType period_type \(.*\);+ValueType period_type \1\
  [ (gogoproto.nullable) = false ];+g
