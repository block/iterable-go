package batch

// Message represents a generic request to be processed in a batch operation.
// It serves as a wrapper that carries both the actual data to be sent to Iterable
// and optional metadata that can be used for tracking or response correlation.
//
// Usage Example:
//
//	message := batch.Message{
//	    Data:     userUpdateRequest,  // The actual API request data
//	    MetaData: "user-123",         // Optional tracking identifier
//	}
//	processor.Add(message)
type Message struct {
	// Data contains the actual request payload to be sent to Iterable
	// (e.g., user data, event data, etc.)
	Data any
	// MetaData holds optional contextual information that
	// can be used for tracking, correlation, or response handling
	MetaData any
}

// Response represents the result of processing a batch request, containing both
// successful results and error information. It maintains a reference to the original
// request for correlation and includes retry information for error handling.
//
// Usage Example:
//
//	// Successful response
//	response := batch.Response{
//	    Data:        apiResponseData,
//	    OriginalReq: originalMessage,
//	    Error:       nil,
//	    Retry:       false,
//	}
//
//	// Failed response that should be retried
//	response := batch.Response{
//	    Data:        nil,
//	    OriginalReq: originalMessage,
//	    Error:       networkError,
//	    Retry:       true,
//	}
type Response struct {
	// Data contains the successful response data from the API operation
	// or nil if error occurred
	Data any
	// OriginalReq holds a reference to the original Message that was processed
	OriginalReq Message
	// Error contains any error that occurred during processing
	// or nil if successful
	Error error
	// Retry indicates whether this failed request should be retried
	// (only relevant when Error is not nil)
	Retry bool
}

// Handler defines the contract for implementing batch processing logic
// for specific API operations. Implementations should handle both batch processing
// (for efficiency) and individual processing (for retry scenarios).
// The handler is responsible for making actual API calls and converting
// results into Response objects.
//
// Usage Example:
//
//	type MyHandler struct {
//	    apiClient *api.Events
//	}
//
//	func (h *MyHandler) ProcessBatch(messages []Message) ([]Response, error, bool) {
//	    // Convert messages to types.EventTrackBulkRequest
//	    results, err := h.apiClient.TrackBulk(batch)
//	    if err != nil {
//	        return nil, err, true // error, can retry
//	    }
//	    // Convert results to Response objects
//	    return responses, nil, false
//	}
//
//	func (h *MyHandler) ProcessOne(message Message) Response {
//	    // Convert message to types.EventTrackRequest
//	    result, err := h.apiClient.Track(request)
//	    return Response{Data: result, OriginalReq: message, Error: err}
//	}
type Handler interface {
	// ProcessBatch processes multiple messages in a single batch operation
	// for efficiency. Returns a slice of Response objects (one per input message),
	// an error if the entire batch failed, and a boolean indicating
	// if the batch operation can be retried.
	ProcessBatch(batch []Message) ([]Response, error, bool)

	// ProcessOne processes a single message individually, typically used
	// for retry scenarios when batch processing fails or when individual messages
	// need special handling.
	// Returns a single Response object with the result of processing the message.
	ProcessOne(message Message) Response
}
