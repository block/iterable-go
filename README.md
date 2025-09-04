# Iterable Go Client Library

A Go (Golang) client library for integrating your applications with the [Iterable](https://www.iterable.com) API.

This library follows the [Iterable API public documentation](https://api.iterable.com/api/docs) and provides both individual API calls and efficient batch processing capabilities.

## Overview

[Iterable](https://www.iterable.com) does not provide an official Go (Golang) implementation of their SDK. While other client libraries are available for different programming languages at [https://github.com/iterable](https://github.com/iterable), Go developers needed better support.

Teams at Block use [iterable-go](https://github.com/block/iterable-go) library to send billions of messages to the Iterable API every day.

The library provides a comprehensive interface to interact with Iterable's marketing automation platform.

It supports:

- **Individual API operations** - Direct calls to Iterable's REST API endpoints
- **Batch processing** - Efficient bulk operations with automatic batching, retries, and error handling
- **Custom configuration** - Flexible configuration options for timeouts, logging, and retry strategies
- **Type safety** - Strongly typed request and response structures



## Using the library

### Basic Usage

Here's a simple example to get started:

```go
package main

import (
    "fmt"
    "log"
    
    iterable_go "github.com/block/iterable-go"
)

func main() {
    // Create a new Iterable client
    client := iterable_go.NewClient("your-api-key")
    
    // Get all campaigns
    campaigns, err := client.Campaigns().All()
    if err != nil {
        log.Fatalf("Error getting campaigns: %v", err)
    }
    
    fmt.Printf("Found %d campaigns\n", len(campaigns))
    for _, campaign := range campaigns {
        fmt.Printf("Campaign (ID: %d): %s \n", campaign.Id, campaign.Name)
    }
}
```

### More Examples

For examples covering all API endpoints and batch processing capabilities, see the [examples](./examples).

It includes:

- **API Examples** - Individual API calls for users, events, campaigns, lists, etc.
- **Batch Examples** - Efficient bulk processing with automatic batching
- **Custom Implementations** - Custom logger and retry strategy examples

### Logging

By default, the library does not log any internal information.
However, if you want to enable logging for debugging, monitoring, or troubleshooting purposes,
you can provide a custom logger that implements the `logger.Logger` interface.

For a more advanced logger implementation with colors and structured output, see [examples/logger_custom.go](./examples/logger_custom.go).

### Configuration

#### NewClient Configuration Options

The `NewClient` function accepts various configuration options to customize the HTTP client behavior:

| Option | Type | Default | Description | Example |
|--------|------|---------|-------------|---------|
| `WithTimeout` | `time.Duration` | `10 * time.Second` | HTTP request timeout | `iterable_go.WithTimeout(30*time.Second)` |
| `WithTransport` | `http.RoundTripper` | `http.DefaultTransport` | Custom HTTP transport | `iterable_go.WithTransport(customTransport)` |
| `WithLogger` | `logger.Logger` | `logger.Noop{}` | Custom logger implementation | `iterable_go.WithLogger(myLogger)` |

**Example:**
```go
client := iterable_go.NewClient(
    "your-api-key",
    iterable_go.WithTimeout(30*time.Second),
    iterable_go.WithLogger(myLogger),
)
```

#### NewBatch Configuration Options

The `NewBatch` function provides extensive configuration for batch processing behavior:

| Option | Type                    | Default            | Description                                      | Example                                              |
|--------|-------------------------|--------------------|--------------------------------------------------|------------------------------------------------------|
| `WithBatchFlushQueueSize` | `int`                   | `100`              | Maximum messages before triggering batch         | `iterable_go.WithBatchFlushQueueSize(50)`            |
| `WithBatchFlushInterval` | `time.Duration`         | `5 * time.Second`  | Maximum time before triggering batch             | `iterable_go.WithBatchFlushInterval(10*time.Second)` |
| `WithBatchBufferSize` | `int`                   | `500`              | Internal channel buffer size                     | `iterable_go.WithBatchBufferSize(1000)`              |
| `WithBatchRetryTimes` | `int`                   | `1`                | Maximum retry attempts                           | `iterable_go.WithBatchRetryTimes(3)`                 |
| `WithBatchSendIndividual` | `bool`                  | `true`             | Send messages individually after a batch failure | `iterable_go.WithBatchSendIndividual(false)`         |
| `WithBatchRetry` | `retry.Retry`           | `ExponentialRetry` | Custom retry strategy                            | `iterable_go.WithBatchRetry(myRetry)`                |
| `WithBatchSendAsync` | `bool`                  | `true`             | Enable asynchronous processing                   | `iterable_go.WithBatchSendAsync(false)`              |
| `WithBatchLogger` | `logger.Logger`         | `logger.Noop{}`    | Custom logger for batch operations               | `iterable_go.WithBatchLogger(myLogger)`              |
| `WithBatchResponseListener` | `chan<- batch.Response` | `nil`              | Channel for response monitoring                  | `iterable_go.WithBatchResponseListener(respChan)`    |

**Example:**
```go
batchClient := iterable_go.NewBatch(
    client,
    iterable_go.WithBatchFlushQueueSize(50),
    iterable_go.WithBatchFlushInterval(10*time.Second),
    iterable_go.WithBatchRetryTimes(3),
    iterable_go.WithBatchLogger(myLogger),
)
```


## License

This project is licensed under the terms specified in [LICENSE](./LICENSE).

---

## Quick Links

- [Iterable Website](https://www.iterable.com)
- [Iterable API Documentation](https://api.iterable.com/api/docs)
- [Other Iterable Client Libraries](https://github.com/iterable)
- [Examples Directory](./examples)
- [License](./LICENSE)
