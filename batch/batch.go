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
//	func (h *MyHandler) ProcessBatch(messages []Message) (ProcessBatchResponse, error) {
//	    // Convert messages to types.EventTrackBulkRequest
//	    results, err := h.apiClient.TrackBulk(batch)
//	    if err != nil {
//	        return batch.StatusRetryIndividual{results}, err // error, can retry
//	    }
//	    // Convert results to Response objects
//	    return batch.StatusSuccess{results}, nil
//	}
//
//	func (h *MyHandler) ProcessOne(message Message) Response {
//	    // Convert message to types.EventTrackRequest
//	    result, err := h.apiClient.Track(request)
//	    return Response{Data: result, OriginalReq: message, Error: err, Retry: err != nil}
//	}
type Handler interface {
	// ProcessBatch processes multiple messages in a single batch operation
	// for efficiency. Returns a slice of Response objects (one per input message),
	// an error if the entire batch failed, and a boolean indicating
	// if the batch operation can be retried.
	ProcessBatch(batch []Message) (ProcessBatchResponse, error)

	// ProcessOne processes a single message individually, typically used
	// for retry scenarios when batch processing fails or when individual messages
	// need special handling.
	// Returns a single Response object with the result of processing the message.
	ProcessOne(message Message) Response
}

// ProcessBatchResponse is an interface that defines the contract for different batch processing outcomes.
// It provides a unified way to handle various response scenarios (success, retry, partial success, etc.)
// while encapsulating the actual response data. Each implementation represents a specific outcome state
// and contains the appropriate response data and any relevant error information.
// The interface method response() returns the slice of Response objects for the batch operation.
type ProcessBatchResponse interface {
	response() []Response
}

// StatusSuccess represents a completely successful batch operation where all messages
type StatusSuccess struct {
	Response []Response
}

// StatusRetryIndividual indicates that the batch operation failed in a way that requires
// retrying each message individually. This typically occurs when the batch operation
// cannot determine the success/failure status of individual messages.
// For example - when Server returns 413:ContentTooLarge, we must retry sending
// the messages individually.
type StatusRetryIndividual struct {
	Response []Response
}

// StatusRetryBatch indicates that the entire batch operation should be retried as a single unit.
// This typically occurs due to temporary issues like network problems or rate limiting.
type StatusRetryBatch struct {
	Response []Response
	BatchErr error
}

// StatusCannotRetry represents a permanent failure state where retrying would not help.
// This occurs in cases like authentication failures, malformed requests
// or batch requests where All non-batch messages failed client validation.
type StatusCannotRetry struct {
	Response []Response
}

// StatusPartialSuccess indicates that some messages in the batch were processed successfully
// while others failed. The Response slice contains a mix of successful and failed results,
// allowing for granular handling of the partial success scenario.
type StatusPartialSuccess struct {
	Response []Response
}

var _ ProcessBatchResponse = StatusSuccess{}
var _ ProcessBatchResponse = StatusRetryIndividual{}
var _ ProcessBatchResponse = StatusRetryBatch{}
var _ ProcessBatchResponse = StatusCannotRetry{}
var _ ProcessBatchResponse = StatusPartialSuccess{}

func (s StatusSuccess) response() []Response         { return s.Response }
func (s StatusRetryIndividual) response() []Response { return s.Response }
func (s StatusRetryBatch) response() []Response      { return s.Response }
func (s StatusCannotRetry) response() []Response     { return s.Response }
func (s StatusPartialSuccess) response() []Response  { return s.Response }
