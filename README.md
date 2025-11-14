# Iterable Go Client Library

[![CI](https://github.com/block/iterable-go/actions/workflows/ci.yml/badge.svg)](https://github.com/block/iterable-go/actions/workflows/ci.yml)

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

| Option          | Type                | Default                 | Description                  | Example                                      |
|-----------------|---------------------|-------------------------|------------------------------|----------------------------------------------|
| `WithTimeout`   | `time.Duration`     | `10 * time.Second`      | HTTP request timeout         | `iterable_go.WithTimeout(30*time.Second)`    |
| `WithTransport` | `http.RoundTripper` | `http.DefaultTransport` | Custom HTTP transport        | `iterable_go.WithTransport(customTransport)` |
| `WithLogger`    | `logger.Logger`     | `logger.Noop{}`         | Custom logger implementation | `iterable_go.WithLogger(myLogger)`           |

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

| Option                               | Type                    | Default            | Description                                           | Example                                              |
|--------------------------------------|-------------------------|--------------------|-------------------------------------------------------|------------------------------------------------------|
| `WithBatchFlushQueueSize`            | `int`                   | `100`              | Maximum messages before triggering batch              | `iterable_go.WithBatchFlushQueueSize(50)`            |
| `WithBatchFlushInterval`             | `time.Duration`         | `5 * time.Second`  | Maximum time before triggering batch                  | `iterable_go.WithBatchFlushInterval(10*time.Second)` |
| `WithBatchBufferSize`                | `int`                   | `500`              | Internal channel buffer size                          | `iterable_go.WithBatchBufferSize(1000)`              |
| `WithBatchRetryTimes`                | `int`                   | `1`                | Maximum retry attempts                                | `iterable_go.WithBatchRetryTimes(3)`                 |
| `WithBatchSendIndividual`            | `bool`                  | `true`             | Send messages individually after a batch failure      | `iterable_go.WithBatchSendIndividual(false)`         |
| `WithBatchNumOfIndividualGoroutines` | `int`                   | `1`                | How manu goroutines to use to send individual retries | `iterable_go.WithBatchNumOfIndividualGoroutines(10)` |
| `WithBatchRetry`                     | `retry.Retry`           | `ExponentialRetry` | Custom retry strategy                                 | `iterable_go.WithBatchRetry(myRetry)`                |
| `WithBatchSendAsync`                 | `bool`                  | `true`             | Enable asynchronous processing                        | `iterable_go.WithBatchSendAsync(false)`              |
| `WithBatchLogger`                    | `logger.Logger`         | `logger.Noop{}`    | Custom logger for batch operations                    | `iterable_go.WithBatchLogger(myLogger)`              |
| `WithBatchResponseListener`          | `chan<- batch.Response` | `nil`              | Channel for response monitoring                       | `iterable_go.WithBatchResponseListener(respChan)`    |

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


## Supported APIs

The following table shows which [Iterable API endpoints](https://api.iterable.com/api/docs) are currently supported by this Go client library:

|                                                           | Endpoint                                                              | Supported | Batch |
|-----------------------------------------------------------|-----------------------------------------------------------------------|-----------|-------|
| **Campaigns**                                             |                                                                       |           |       |
| List campaign metadata                                    | GET `/api/campaigns`                                                  | ✅         |       |
| Abort campaign                                            | POST `/api/campaigns/abort`                                           | ✅         |       |
| Activate a triggered campaign                             | POST `/api/campaigns/activateTriggered`                               | ✅         |       |
| Archive campaigns                                         | POST `/api/campaigns/archive`                                         |           |       |
| Cancel a scheduled or <br/>recurring campaign             | POST `/api/campaigns/cancel`                                          | ✅         |       |
| Create a campaign                                         | POST `/api/campaigns/create`                                          | ✅         |       |
| Deactivate a triggered <br/>campaign                      | POST `/api/campaigns/deactivateTriggered`                             |           |       |
| Get metrics for campaigns                                 | GET `/api/campaigns/metrics`                                          |           |       |
| Get child campaigns of <br/>a recurring campaign          | GET `/api/campaigns/recurring/{id}/childCampaigns`                    | ✅         |       |
| Trigger a campaign                                        | POST `/api/campaigns/trigger`                                         | ✅         |       |
| Schedule existing campaign <br/>to be sent                | POST `/api/campaigns/{id}/schedule`                                   |           |       |
| Send existing campaign now                                | POST `/api/campaigns/{id}/send`                                       |           |       |
| Get a campaign                                            | GET `/api/campaigns/{id}`                                             |           |       |
|                                                           |
| **Catalogs**                                              |                                                                       |           |       |
| Get catalog names                                         | GET `/api/catalogs`                                                   | ✅         |       |
| Delete a catalog                                          | DELETE `/api/catalogs/{catalogName}`                                  |           |       |
| Create a catalog                                          | POST `/api/catalogs/{catalogName}`                                    |           |       |
| Get field mappings for a catalog                          | GET `/api/catalogs/{catalogName}/fieldMappings`                       | ✅         |       |
| Set a catalog's field mappings (data types)               | PUT `/api/catalogs/{catalogName}/fieldMappings`                       |           |       |
| Bulk delete catalog items                                 | DELETE `/api/catalogs/{catalogName}/items`                            |           |       |
| Get the catalog items for a catalog                       | GET `/api/catalogs/{catalogName}/items`                               |           |       |
| Bulk create catalog items                                 | POST `/api/catalogs/{catalogName}/items`                              |           |       |
| Delete a catalog item                                     | DELETE `/api/catalogs/{catalogName}/items/{itemId}`                   |           |       |
| Get a specific catalog item                               | GET `/api/catalogs/{catalogName}/items/{itemId}`                      |           |       |
| Create or update a catalog item                           | PATCH `/api/catalogs/{catalogName}/items/{itemId}`                    |           |       |
| Create or replace a catalog item                          | PUT `/api/catalogs/{catalogName}/items/{itemId}`                      |           |       |
|                                                           |
| **Channels**                                              |
| Get channels                                              | GET `/api/channels`                                                   | ✅         |       |
|                                                           |
| **Commerce**                                              |
| Track a purchase                                          | POST `/api/commerce/trackPurchase`                                    |           |       |
| Update a user's shopping cart items                       | POST `/api/commerce/updateCart`                                       |           |       |
|                                                           |
| **Email**                                                 |
| Cancel an email to a user                                 | POST `/api/email/cancel`                                              |           |       |
| Send an email to an email address                         | POST `/api/email/target`                                              |           |       |
| View a previously sent email                              | GET `/api/email/viewInBrowser`                                        |           |       |
|                                                           |
| **Embedded Messaging**                                    |
| Get a user's embedded messages                            | GET `/api/embedded-messaging/messages`                                |           |       |
|                                                           |
| **Events**                                                |
| Track an embedded message click                           | POST `/api/embedded-messaging/events/click`                           |           |       |
| Track an embedded message received event                  | POST `/api/embedded-messaging/events/received`                        |           |       |
| Track an embedded message session and related impressions | POST `/api/embedded-messaging/events/session`                         |           |       |
| Get user events by userId                                 | GET `/api/events/byUserId/{userId}`                                   | ✅         |       |
| Consume or delete an in-app message                       | POST `/api/events/inAppConsume`                                       |           |       |
| Track an event                                            | POST `/api/events/track`                                              | ✅         |       |
| Bulk track events                                         | POST `/api/events/trackBulk`                                          | ✅         | ✅     |
| Track an in-app message click                             | POST `/api/events/trackInAppClick`                                    |           |       |
| Track the closing of an in-app message                    | POST `/api/events/trackInAppClose`                                    |           |       |
| Track the delivery of an in-app message                   | POST `/api/events/trackInAppDelivery`                                 |           |       |
| Track an in-app message open                              | POST `/api/events/trackInAppOpen`                                     |           |       |
| Track a mobile push open                                  | POST `/api/events/trackPushOpen`                                      |           |       |
| Track a web push click                                    | POST `/api/events/trackWebPushClick`                                  |           |       |
| Get user events                                           | GET `/api/events/{email}`                                             | ✅         |       |
|                                                           |
| **Experiments**                                           |
| Get metrics for experiments                               | GET `/api/experiments/metrics`                                        |           |       |
|                                                           |
| **Export**                                                |
| Export data to CSV                                        | GET `/api/export/data.csv`                                            |           |       |
| Export data to JSON                                       | GET `/api/export/data.json`                                           |           |       |
| Get export jobs                                           | GET `/api/export/jobs`                                                |           |       |
| Start export                                              | POST `/api/export/start`                                              |           |       |
| Export user events                                        | GET `/api/export/userEvents`                                          |           |       |
| Cancel export                                             | DELETE `/api/export/{jobId}`                                          |           |       |
| Get export files                                          | GET `/api/export/{jobId}/files`                                       |           |       |
|                                                           |
| **In-App**                                                |
| Cancel a scheduled in-app message                         | POST `/api/inApp/cancel`                                              |           |       |
| Get a user's in-app messages                              | GET `/api/inApp/getMessages`                                          |           |       |
| Get a user's most relevant in-app message                 | GET `/api/inApp/getPriorityMessage`                                   |           |       |
| Send an in-app notification to a user                     | POST `/api/inApp/target`                                              |           |       |
|                                                           |
| **In-App-Web**                                            |
| Get a user's web in-app messages                          | GET `/api/inApp/web/getMessages`                                      |           |       |
|                                                           |
| **Lists**                                                 |
| Get lists                                                 | GET `/api/lists`                                                      | ✅         |       |
| Create a static list                                      | POST `/api/lists`                                                     | ✅         |       |
| Get users in a list                                       | GET `/api/lists/getUsers`                                             | ✅         |       |
| Preview users in a list                                   | GET `/api/lists/previewUsers`                                         |           |       |
| Add subscribers to list                                   | POST `/api/lists/subscribe`                                           | ✅         | ✅     |
| Remove users from a list                                  | POST `/api/lists/unsubscribe`                                         | ✅         | ✅     |
| Delete a list                                             | DELETE `/api/lists/{listId}`                                          | ✅         |       |
| Get count of users in list                                | GET `/api/lists/{listId}/size`                                        | ✅         |       |
|                                                           |
| **MessageTypes**                                          |
| List message types                                        | GET `/api/messageTypes`                                               | ✅         |       |
|                                                           |
| **Metadata**                                              |
| List available tables                                     | GET `/api/metadata`                                                   |           |       |
| Delete a table                                            | DELETE `/api/metadata/{table}`                                        |           |       |
| List keys in a table                                      | GET `/api/metadata/{table}`                                           |           |       |
| Delete a single metadata key/value                        | DELETE `/api/metadata/{table}/{key}`                                  |           |       |
| Get the metadata value of a single key                    | GET `/api/metadata/{table}/{key}`                                     |           |       |
| Create or replace metadata                                | PUT `/api/metadata/{table}/{key}`                                     |           |       |
|                                                           |
| **Push**                                                  |
| Cancel a push notification to a user                      | POST `/api/push/cancel`                                               |           |       |
| Send push notification to user                            | POST `/api/push/target`                                               |           |       |
|                                                           |
| **SMS**                                                   |
| Cancel an SMS to a user                                   | POST `/api/sms/cancel`                                                |           |       |
| Send SMS notification to user                             | POST `/api/sms/target`                                                |           |       |
|                                                           |
| **Snippets**                                              |
| Get all snippets                                          | GET `/api/snippets`                                                   |           |       |
| Create a snippet                                          | POST `/api/snippets`                                                  |           |       |
| Delete a snippet                                          | DELETE `/api/snippets/{identifier}`                                   |           |       |
| Get snippet by ID or name                                 | GET `/api/snippets/{identifier}`                                      |           |       |
| Create or update a snippet                                | PUT `/api/snippets/{identifier}`                                      |           |       |
|                                                           |
| **Subscriptions**                                         |
| Trigger a double opt-in subscription flow                 | POST `/api/subscriptions/subscribeToDoubleOptIn`                      |           |       |
| Bulk subscription action on a list of users               | PUT `/api/subscriptions/{subGroup}/{subGroupId}`                      |           |       |
| Unsubscribe a single user by userId                       | DELETE `/api/subscriptions/{subGroup}/{subGroupId}/byUserId/{userId}` |           |       |
| Subscribe a single user by their userId                   | PATCH `/api/subscriptions/{subGroup}/{subGroupId}/byUserId/{userId}`  |           |       |
| Unsubscribe a single user                                 | DELETE `/api/subscriptions/{subGroup}/{subGroupId}/user/{userEmail}`  |           |       |
| Subscribe a single user                                   | GET `/api/subscriptions/{subGroup}/{subGroupId}/user/{userEmail}`     |           |       |
|                                                           |
| **Templates**                                             |
| Get project templates                                     | GET `/api/templates`                                                  |           |       |
| Bulk delete templates                                     | POST `/api/templates/bulkDelete`                                      |           |       |
| Get an email template by templateId                       | GET `/api/templates/email/get`                                        |           |       |
| Update email template                                     | POST `/api/templates/email/update`                                    |           |       |
| Create email template                                     | POST `/api/templates/email/upsert`                                    |           |       |
| Get an email template by clientTemplateId                 | GET `/api/templates/getByClientTemplateId`                            |           |       |
| Get an in-app template                                    | GET `/api/templates/inapp/get`                                        |           |       |
| Update in-app template                                    | POST `/api/templates/inapp/update`                                    |           |       |
| Create an in-app template                                 | POST `/api/templates/inapp/upsert`                                    |           |       |
| Get a push template                                       | GET `/api/templates/push/get`                                         |           |       |
| Update push template                                      | POST `/api/templates/push/update`                                     |           |       |
| Create a push template                                    | POST `/api/templates/push/upsert`                                     |           |       |
| Get an SMS template                                       | GET `/api//templates/sms/get`                                         |           |       |
| Update SMS template                                       | POST `/api/templates/sms/update`                                      |           |       |
| Create an SMS template                                    | POST `/api/templates/sms/upsert`                                      |           |       |
|                                                           |
| **Users**                                                 |
| Invalidate all JWTs issued for a user                     | POST `/api/auth/jwts/invalidate`                                      |           |       |
| Bulk update user data                                     | POST `/api/users/bulkUpdate`                                          | ✅         | ✅     |
| Bulk update user subscriptions                            | POST `/api/users/bulkUpdateSubscriptions`                             | ✅         | ✅     |
| Get a user by userId (query parameter)                    | GET `/api/users/byUserId`                                             | ✅         |       |
| Delete user by userId                                     | DELETE `/api/users/byUserId/{userId}`                                 | ✅         |       |
| Get a user by userId (path parameter)                     | GET `/api/users/byUserId/{userId}`                                    |           |       |
| Disable pushes to a mobile device                         | POST `/api/users/disableDevice`                                       |           |       |
| Forget a user in compliance with GDPR                     | POST `/api/users/forget`                                              | ✅         |       |
| Get hashed forgotten users in compliance with GDPR        | GET `/api/users/forgotten`                                            | ✅         |       |
| Get hashed forgotten userIds in compliance with GDPR      | GET `/api/users/forgottenUserIds`                                     |           |       |
| Get a user by email (query parameter)                     | GET `/api/users/getByEmail`                                           | ✅         |       |
| Get all user fields                                       | GET `/api/users/getFields`                                            | ✅         |       |
| Get messages sent to a user                               | GET `/api/users/getSentMessages`                                      | ✅         |       |
| Merge users                                               | POST `/api/users/merge`                                               |           |       |
| Register a browser token for web push                     | POST `/api/users/registerBrowserToken`                                |           |       |
| Register a device token for push                          | POST `/api/users/registerDeviceToken`                                 |           |       |
| Unforget a user in compliance with GDPR                   | POST `/api/users/unforget`                                            | ✅         |       |
| Update user data                                          | POST `/api/users/update`                                              | ✅         |       |
| Update user email                                         | POST `/api/users/updateEmail`                                         | ✅         |       |
| Update user subscriptions                                 | POST `/api/users/updateSubscriptions`                                 | ✅         |       |
| Delete a user by email                                    | DELETE `/api/users/{email}`                                           | ✅         |       |
| Get a user by email (path parameter)                      | GET `/api/users/{email}`                                              |           |       |
|                                                           |
| **Verify**                                                |
| Begin SMS Verification                                    | POST `/api/verify/sms/begin`                                          |           |       |
| Check SMS Verification Code                               | POST `/api/verify/sms/check`                                          |           |       |
|                                                           |
| **Webhooks**                                              |
| Get webhooks                                              | GET `/api/webhooks`                                                   |           |       |
| Update webhook                                            | POST `/api/webhooks`                                                  |           |       |
|                                                           |
| **WebPush**                                               |
| Cancel a web push notification to a user                  | POST `/api/webPush/cancel`                                            |           |       |
| Send web push notification to user                        | POST `/api/webPush/target`                                            |           |       |
|                                                           |
| **WhatsApp**                                              |
| Cancel a scheduled WhatsApp message                       | POST `/api/whatsApp/cancel`                                           |           |       |
| Send a WhatsApp message to a user                         | POST `/api/whatsApp/target`                                           |           |       |
|                                                           |
| **Workflows**                                             |
| Get journeys (workflows)                                  | GET `/api/journeys`                                                   |           |       |
| Trigger a journey (workflow)                              | POST `/api/workflows/triggerWorkflow`                                 |           |       |

## License

This project is licensed under the terms specified in [LICENSE](./LICENSE).

---

## Quick Links

- [Iterable Website](https://www.iterable.com)
- [Iterable API Documentation](https://api.iterable.com/api/docs)
- [Other Iterable Client Libraries](https://github.com/iterable)
- [Examples Directory](./examples)
- [License](./LICENSE)
