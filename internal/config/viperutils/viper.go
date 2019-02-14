package viperutils

import (
	"bytes"

	"github.com/spf13/viper"
)

// ViperFromYAMLBytes unmarshals byte content in the YAML file format into
// a viper object
func ViperFromYAMLBytes(yamlBlob []byte) (*viper.Viper, error) {
	v := viper.New()
	v.SetConfigType("yaml")
	if err := v.ReadConfig(bytes.NewBuffer(yamlBlob)); err != nil {
		return nil, err
	}
	return v, nil
}
