# Iterable Go Client Examples

This directory contains examples demonstrating how to use the Iterable Go client library.
Each example shows how to use different API endpoints and features.

## Setup

Before running any examples, you need to:

1. Replace `__API_KEY__` in `main.go` file with your actual Iterable API key
2. Update any placeholder values (like email addresses, user IDs, list IDs) with real values from your Iterable account

## Running Examples

The `main.go` file contains calls to all example functions.
By default, only the first example is uncommented:

```bash
cd examples
go run .
```

To run different examples, edit `main.go` and uncomment the desired function calls.

## Error Handling

All examples include basic error handling.
In production code, you should implement more robust error handling based on your specific needs.

## Configuration

The examples use the default client configuration.
You can customize the client with options like:

```go
client := iterable_go.NewClient(
    apiKey,
    iterable_go.WithTimeout(30*time.Second),
    iterable_go.WithLogger(myLogger),
)
```

Refer to the main documentation for more configuration options.
