package configmapprovider

import (
	"context"

	"go.opentelemetry.io/collector/config"
)

// ConfigSource is an interface that helps to retrieve a config map and watch for any
// changes to the config map. Implementations may load the config from a file,
// a database or any other source.
type ConfigSource interface {
	Shutdownable

	// Retrieve goes to the configuration source and retrieves the selected data which
	// contains the value to be injected in the configuration and the corresponding watcher that
	// will be used to monitor for updates of the retrieved value.
	//
	// onChange callback is called when the config changes. onChange may be called from
	// a different go routine. After onChange is called Retrieved.Get should be called
	// to get the new config. See description of Retrieved for more details.
	// onChange may be nil, which indicates that the caller is not interested in
	// knowing about the changes.
	//
	// If ctx is cancelled should return immediately with an error.
	// Should never be called concurrently with itself or with Shutdown.
	Retrieve(ctx context.Context, onChange func(*ChangeEvent)) (RetrievedConfig, error)
}

// RetrievedConfig holds the result of a call to the Retrieve method of a ConfigSource object.
//
// The typical usage is the following:
//
//		r := mapProvider.Retrieve()
//		r.Get()
//		// wait for onChange() to be called.
//		r.Close()
//		r = mapProvider.Retrieve()
//		r.Get()
//		// wait for onChange() to be called.
//		r.Close()
//		// repeat Retrieve/Get/wait/Close cycle until it is time to shut down the Collector process.
//		// ...
//		mapProvider.Shutdown()
type RetrievedConfig interface {
	// Get returns the config Map.
	// If Close is called before Get or concurrently with Get then Get
	// should return immediately with ErrSessionClosed error.
	// Should never be called concurrently with itself.
	// If ctx is cancelled should return immediately with an error.
	Get(ctx context.Context) (*config.Map, error)

	// Close signals that the configuration for which it was used to retrieve values is
	// no longer in use and the object should close and release any watchers that it
	// may have created.
	//
	// This method must be called when the service ends, either in case of success or error.
	//
	// Should never be called concurrently with itself.
	// May be called before, after or concurrently with Get.
	// If ctx is cancelled should return immediately with an error.
	//
	// Calling Close on an already closed object should have no effect and should return nil.
	Close(ctx context.Context) error
}
