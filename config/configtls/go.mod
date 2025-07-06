module go.opentelemetry.io/collector/config/configtls

go 1.23.0

require (
	github.com/foxboron/go-tpm-keyfiles v0.0.0-20250323135004-b31fac66206e
	github.com/fsnotify/fsnotify v1.9.0
	github.com/google/go-tpm v0.9.5
	github.com/stretchr/testify v1.10.0
	go.opentelemetry.io/collector/config/configopaque v1.35.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/google/go-tpm-tools v0.4.4 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/crypto v0.35.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace go.opentelemetry.io/collector/config/configopaque => ../configopaque
