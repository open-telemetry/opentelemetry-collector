package infoscraper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestCreateMetricsScraper(t *testing.T) {
	factory := &Factory{}
	scraper, err := factory.CreateMetricsScraper(context.Background(), zap.NewNop(), nil)

	assert.NoError(t, err)
	assert.NotNil(t, scraper)
}

func TestCreateMetricsScraper_Error(t *testing.T) {
	factory := &Factory{}

	_, err := factory.CreateMetricsScraper(context.Background(), zap.NewNop(), nil)

	assert.Nil(t, err)
}
