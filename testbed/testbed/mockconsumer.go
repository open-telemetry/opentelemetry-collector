package testbed

// Mock_DataConsumer is an interface that
//keeps the count of number of events received by mock receiver.
//This is mainly usefull for the Exporters that are not have the matching receiver
type Mock_DataConsumer interface {
	// MockConsumeData receives metrics and traces and just count the neumber of event received.
	Mock_ConsumeData(spansCount int) error
}
