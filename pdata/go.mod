module go.opentelemetry.io/collector/pdata

go 1.25.0

require (
	github.com/json-iterator/go v1.1.12
	github.com/stretchr/testify v1.12.0
	go.opentelemetry.io/collector/featuregate v1.65.0
	go.opentelemetry.io/collector/internal/testutil v0.159.0
	go.opentelemetry.io/proto/slim/otlp v1.11.0
	go.opentelemetry.io/proto/slim/otlp/collector/profiles/v1development v0.4.0
	go.opentelemetry.io/proto/slim/otlp/profiles/v1development v0.4.0
	go.uber.org/goleak v1.3.0
	go.uber.org/multierr v1.11.0
	google.golang.org/grpc v1.83.0
	google.golang.org/protobuf v1.36.12
)

require (
	github.com/hashicorp/go-version v1.9.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	golang.org/x/net v0.55.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/text v0.40.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260526163538-3dc84a4a5aaa // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

retract (
	v1.0.0-rc10 // RC version scheme discovered to be alphabetical, use v1.0.0-rcv0011 instead
	v0.57.1 // Release failed, use v0.57.2
	v0.57.0 // Release failed, use v0.57.2
)

replace go.opentelemetry.io/collector/featuregate => ../featuregate

replace go.opentelemetry.io/collector/internal/testutil => ../internal/testutil
