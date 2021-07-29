package s3configsource

import (
	splunkprovider "github.com/signalfx/splunk-otel-collector"
)

type Config struct {

	//AWS variables that likely won't be used but nice to have in case of future modification
	*splunkprovider.Settings

	Region string `mapstructure:"region"`

	Bucket string `mapstructure:"bucket"`

	Key string `mapstructure:"key"`

	VersionId string `mapstructure:"version_id"`
}
