// Code generated from Pkl module `BatchProcessorConfig`. DO NOT EDIT.
package batchprocessor

import "github.com/apple/pkl-go/pkl"

type Config struct {
	Timeout *pkl.Duration `pkl:"timeout" mapstructure:"timeout"`

	SendBatchSize uint32 `pkl:"send_batch_size" mapstructure:"send_batch_size"`

	SendBatchMaxSize uint32 `pkl:"send_batch_max_size" mapstructure:"send_batch_max_size"`

	MetadataKeys []string `pkl:"metadata_keys" mapstructure:"metadata_keys"`

	MetadataCardinalityLimit uint32 `pkl:"metadata_cardinality_limit" mapstructure:"metadata_cardinality_limit"`
}
