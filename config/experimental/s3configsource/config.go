package s3configsource

import (
	"github.com/knadh/koanf"
	splunkprovider "github.com/signalfx/splunk-otel-collector"
)

type Config struct {
	//koanf has a nice YAML parser and is already being used in OTEL - no particular dependence on it
	//Will use this just to parse the YAML and find specific keys in retrieve
	InternalConfig *koanf.Koanf

	//AWS variables that likely won't be used but nice to have in case of future modification
	*splunkprovider.Settings

	Region string `mapstructure:"region"`

	Bucket string `mapstructure:"bucket"`

	Key string `mapstructure:"key"`

	VersionId string `mapstructure:"version_id"`
}
