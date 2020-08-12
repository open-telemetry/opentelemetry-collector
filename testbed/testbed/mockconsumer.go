package testbed

// Mock_DataConsumer is an interface that
//keeps the count of number of events received by mock receiver.
//This is mainly useful for the Exporters that are not have the matching receiver
type MockDataConsumer interface {
	// MockConsumeData receives metrics and traces and just count the number of event received.
	MockConsumeData(spansCount int) error
}
