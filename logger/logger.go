package logger

// Logger provides a standardized logging interface for the Iterable Go client.
// It defines methods for different log levels (Debug, Info, Warn, Error) to enable
// consistent logging throughout the client library. This interface allows users
// to plug in their preferred logging implementation (e.g., glog, logrus, zap, standard log)
// or use the provided Noop logger to disable logging entirely.
//
// The logger is used throughout the client for:
// - API request/response debugging
// - Batch processing status and errors
// - Retry attempt tracking
// - Connection and transport issues
//
// Usage Example:
//
//	// Using with a custom logger implementation
//	client := iterable_go.NewClient(apiKey, iterable_go.WithLogger(myLogger))
//
//	// Using with batch processing
//	batch := iterable_go.NewBatch(client, iterable_go.WithBatchLogger(myLogger))
//
//	// Disable logging entirely
//	client := iterable_go.NewClient(apiKey, iterable_go.WithLogger(&logger.Noop{}))
type Logger interface {
	Debugf(format string, args ...any)
	Infof(format string, args ...any)
	Warnf(format string, args ...any)
	Errorf(format string, args ...any)
}
