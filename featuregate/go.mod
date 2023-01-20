module go.opentelemetry.io/collector/featuregate

go 1.18

require (
	github.com/stretchr/testify v1.8.1
	go.uber.org/atomic v1.10.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

retract v0.69.0 // Release failed, use v0.69.1
