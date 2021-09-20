package parserprovider

import (
	"go.opentelemetry.io/collector/config"
)

// TODO: This probably will make sense to be exported, but needs better name and documentation.
type simpleRetrieved struct {
	confMap *config.Map
}

func (sr *simpleRetrieved) Get() *config.Map {
	return sr.confMap
}
