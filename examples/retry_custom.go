package main

import (
	"fmt"
	"time"

	iterable_go "iterable-go"
	"iterable-go/batch"
	"iterable-go/retry"
	"iterable-go/types"
)

func retry_custom(apiKey string) {
	myRetry := NewMyAwesomeRetry("MyAwesomeRetry")

	fmt.Println("=== Custom Retry Demo ===")
	testAttempts := 3
	err := myRetry.Do(
		testAttempts,
		"demo-function",
		func(attempt int) (error, retry.ExitStrategy) {
			if attempt < 2 { // Fail first 2 attempts
				err := fmt.Errorf("simulated failure on attempt %d", attempt+1)
				return err, retry.Continue
			}
			// Success on 3rd attempt
			return nil, retry.StopNow
		},
	)

	if err != nil {
		fmt.Printf("Demo function failed: %v\n", err)
	} else {
		fmt.Printf("Demo function succeeded!\n")
	}
	fmt.Println()

	// Create a new Iterable client (retry is used internally by batch processors)
	fmt.Println("=== Creating Iterable Client ===")
	client := iterable_go.NewClient(apiKey)
	fmt.Println("✅ Iterable client created successfully")
	fmt.Println()

	// Create a batch client with our custom retry strategy
	fmt.Println("=== Creating Batch Client with Custom Retry ===")
	responseChan := make(chan batch.Response, 100)

	batchClient := iterable_go.NewBatch(client,
		// Use our custom retry strategy for batch operations
		iterable_go.WithBatchRetry(myRetry),
		iterable_go.WithBatchRetryTimes(2),     // Try up to 2 times
		iterable_go.WithBatchFlushQueueSize(1), // Process immediately
		iterable_go.WithBatchFlushInterval(1*time.Second),
		iterable_go.WithBatchResponseListener(responseChan),
	)

	fmt.Println("✅ Batch client created with MyAwesomeRetry strategy")
	fmt.Printf("   - Max retry attempts: 2\n")
	fmt.Printf("   - Retry delay: 5 seconds (custom implementation)\n")
	fmt.Printf("   - Retry strategy: MyAwesomeRetry\n")
	fmt.Println()

	// Start the event tracking processor
	fmt.Println("=== Starting Event Tracking with Custom Retry ===")
	batchClient.EventTrack().Start()

	// Add an event that might trigger retry logic (if API fails)
	eventMessage := batch.Message{
		Data: &types.EventTrackRequest{
			Email:     "test@example.com",
			EventName: "custom_retry_test",
			DataFields: map[string]interface{}{
				"retry_strategy": "MyAwesomeRetry",
				"delay_seconds":  5,
				"test_purpose":   "demonstrate custom retry implementation",
			},
		},
		MetaData: "custom-retry-demo",
	}

	fmt.Println("📤 Adding event to batch processor...")
	batchClient.EventTrack().Add(eventMessage)

	// Listen for responses in a separate goroutine
	go func() {
		for response := range responseChan {
			if response.Error != nil {
				fmt.Printf("❌ Event processing failed: %v (MetaData: %v)\n", response.Error, response.OriginalReq.MetaData)
			} else {
				fmt.Printf("✅ Event processed successfully (MetaData: %v)\n", response.OriginalReq.MetaData)
			}
		}
	}()

	// Wait for processing to complete (and potential retries)
	fmt.Println("⏳ Waiting for batch processing (may see retry logic if API fails)...")
	time.Sleep(15 * time.Second) // Give enough time for retries if needed

	// Stop the processor
	fmt.Println("🛑 Stopping batch processor...")
	batchClient.EventTrack().Stop()
	close(responseChan)

	fmt.Println()
	fmt.Println("=== Custom Retry Implementation Complete ===")
	fmt.Println("✅ MyAwesomeRetry successfully implemented retry.Retry interface")
	fmt.Println("✅ Custom 5-second delay strategy demonstrated")
	fmt.Println("✅ Integration with batch client completed")
	fmt.Println("✅ Retry logic will be used for any failed batch operations")

	// Additional demonstration: Show retry with different scenarios
	fmt.Println()
	fmt.Println("=== Additional Retry Scenarios ===")

	// Scenario 1: Immediate stop
	fmt.Println("Scenario 1: Function requests immediate stop")
	err = myRetry.Do(3, "immediate-stop-test", func(attempt int) (error, retry.ExitStrategy) {
		return fmt.Errorf("critical error - stop immediately"), retry.StopNow
	})
	fmt.Printf("Result: %v\n", err)
	fmt.Println()

	// Scenario 2: All attempts fail
	fmt.Println("Scenario 2: All attempts fail")
	err = myRetry.Do(2, "all-fail-test", func(attempt int) (error, retry.ExitStrategy) {
		return fmt.Errorf("persistent error on attempt %d", attempt+1), retry.Continue
	})
	fmt.Printf("Result: %v\n", err)
	fmt.Println()

	fmt.Println("🎉 Custom retry implementation demo completed!")
}

// MyAwesomeRetry implements the retry.Retry interface
// with a custom 5-second delay strategy
type MyAwesomeRetry struct {
	name string
}

var _ retry.Retry = &MyAwesomeRetry{}

func NewMyAwesomeRetry(name string) *MyAwesomeRetry {
	return &MyAwesomeRetry{
		name: name,
	}
}

func (r *MyAwesomeRetry) Do(attempts int, fnName string, fn retry.RetriableFn) error {
	fmt.Printf(
		"🔄 [%s] Starting retry logic for '%s' with max %d attempts\n",
		r.name, fnName, attempts,
	)

	if attempts < 1 {
		return fmt.Errorf("attempts must be > 0, got %d", attempts)
	}

	var lastErr error

	for attempt := 0; attempt < attempts; attempt++ {
		fmt.Printf("⏳ [%s] Attempt %d/%d for '%s'\n", r.name, attempt+1, attempts, fnName)

		// Execute the retriable function
		err, exitStrategy := fn(attempt)

		// If successful, return immediately
		if err == nil {
			fmt.Printf(
				"✅ [%s] Success on attempt %d/%d for '%s'\n",
				r.name, attempt+1, attempts, fnName,
			)
			return nil
		}

		// Store the error
		lastErr = err

		// If function says to stop immediately, don't retry
		if exitStrategy == retry.StopNow {
			fmt.Printf(
				"🛑 [%s] Stop requested on attempt %d/%d for '%s': %v\n",
				r.name, attempt+1, attempts, fnName, err,
			)
			return err
		}

		// If this was the last attempt, don't sleep
		if attempt == attempts-1 {
			fmt.Printf(
				"❌ [%s] Final attempt %d/%d failed for '%s': %v\n",
				r.name, attempt+1, attempts, fnName, err,
			)
			break
		}

		// Sleep for 5 seconds before next attempt (as requested)
		fmt.Printf(
			"😴 [%s] Sleeping 5 seconds before next attempt for '%s'...\n",
			r.name, fnName,
		)
		time.Sleep(5 * time.Second)
	}

	fmt.Printf(
		"💥 [%s] All %d attempts exhausted for '%s', final error: %v\n",
		r.name, attempts, fnName, lastErr,
	)
	return lastErr
}
