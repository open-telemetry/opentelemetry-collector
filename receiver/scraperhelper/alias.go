package scraperhelper

var (
	AddScraper                   = AddMetricsScraper
	NewScraperControllerReceiver = NewMetricsScraperControllerReceiver
	WithTickerChannel            = WithMetricsTickerChannel
)

type ScraperControllerOption = MetricsScraperControllerOption
