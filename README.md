# Letterbox

Letterbox is a generic Go library for asynchronous event delivery and processing using the Inbox/Outbox pattern.

It provides a common event model and abstractions for persisting, publishing, receiving and processing events while keeping storage and transport concerns decoupled from application logic.

## Concept

```
              ┌─────────────────────────────────────┐
              │                                     │
Application ──► Outbox ──► Transport ──► Inbox ──► Handler
              │                                     │
              └─────────────────────────────────────┘
                    │                       │
                    ▼                       ▼
               delivery state         processing state
                    │                       │
                    └── retry                └── retry
```

Letterbox is built around two durable message flows:

- **Outbox** - stores outgoing events and tracks their delivery state until they are successfully published.
- **Inbox** - stores incoming events before acknowledging their receipt and tracks their processing state until they are successfully handled.

Once an event is persisted, Letterbox coordinates its further delivery or processing through the configured storage and transport adapters.

Applications operate on strongly typed events, while Letterbox uses a generic event envelope internally to exchange and persist them.

## Goals

- Transport-agnostic event processing
- Storage-agnostic Inbox and Outbox
- Reliable delivery and processing building blocks
- Explicit delivery and processing state
- Retry and duplicate-aware storage hooks
- Strongly typed application events
- Minimal integration with application code

## Table of Contents

- [Installation](#installation)
- [Outbox API](#outbox-api)
- [Inbox API](#inbox-api)
- [Event Model](#event-model)
- [Outbox Flow](#outbox-flow)
- [Outbox Retention](#outbox-retention)
- [Inbox Flow](#inbox-flow)
- [Inbox Retention](#inbox-retention)
- [Reliability Model](#reliability-model)
- [Configuration](#configuration)
- [Storage Contracts](#storage-contracts)
- [Examples](#examples)

## Installation

Install the library in an existing Go module with:

```bash
go get github.com/ternaryss/go-letterbox@latest
```

Use the library from the `pkg/letterbox` package:

```go
import "github.com/ternaryss/go-letterbox/pkg/letterbox"
```

## Outbox API

The Outbox API is exposed from the `github.com/ternaryss/go-letterbox/pkg/letterbox` package.

```go
outbox, err := letterbox.NewOutbox(sender, storage, options...)
```

`NewOutbox` requires:

- `sender`: a non-empty identifier of the publishing application.
- `storage`: an implementation of `letterbox.OutboxStore`.
- `options`: `letterbox.OutboxOption` values, including a configured publisher.

Available Outbox methods:

| Method | Description |
| --- | --- |
| `Publish(event Event) error` | Encodes an application event and stores it as a pending message. |
| `Flush() error` | Processes pending messages once. |
| `Start()` | Starts scheduled processing of pending messages. |
| `Stop() error` | Stops the scheduler used by `Start`. |

The built-in publishers are:

- `WithOutboxConsolePublisher()` - logs published envelopes with `log/slog`.
- `WithOutboxRabbitMqPublisher(url, exchange, timeout)` - publishes envelopes to RabbitMQ.

Minimal Outbox setup:

```go
outbox, err := letterbox.NewOutbox(
    "example-app",
    storage,
    letterbox.WithOutboxConsolePublisher(),
)
```

RabbitMQ Outbox setup:

```go
outbox, err := letterbox.NewOutbox(
    "example-app",
    storage,
    letterbox.WithOutboxRabbitMqPublisher(
        "amqp://admin:admin@rabbitmq:5672/",
        "letterbox.events",
        5*time.Second,
    ),
)
```

## Inbox API

The Inbox API stores incoming envelopes, then dispatches stored messages to strongly typed handlers.

```go
inbox, err := letterbox.NewInbox(storage, options...)
```

`NewInbox` requires:

- `storage`: an implementation of `letterbox.InboxStore`.
- `options`: `letterbox.InboxOption` values, including a configured consumer.

Available Inbox methods:

| Method | Description |
| --- | --- |
| `Receive(envelope Envelope) (bool, error)` | Stores an incoming envelope as a received message. |
| `Flush() error` | Processes received messages once. |
| `Start()` | Starts scheduled processing of received messages. |
| `Stop() error` | Stops the scheduler used by `Start` and closes the configured consumer. |

Handlers are registered with `Subscribe`:

```go
err := letterbox.Subscribe(inbox, func(event UserCreated) error {
    // application logic
    return nil
})
```

The built-in consumers are:

- `WithInboxHttpConsumer(mux)` - registers `POST /events` on the provided `*http.ServeMux`.
- `WithInboxRabbitMqConsumer(url, queue)` - consumes envelopes from an existing RabbitMQ queue.

Minimal Inbox setup:

```go
mux := http.NewServeMux()

inbox, err := letterbox.NewInbox(
    storage,
    letterbox.WithInboxHttpConsumer(mux),
)
```

RabbitMQ Inbox setup:

```go
inbox, err := letterbox.NewInbox(
    storage,
    letterbox.WithInboxInterval(time.Second),
    letterbox.WithInboxRabbitMqConsumer(
        "amqp://admin:admin@rabbitmq:5672/",
        "example.inbox",
    ),
)
```

## Event Model

Application events must implement `letterbox.Event`:

```go
type Event interface {
    Type() string
    Version() int
}
```

`Type` is the stable event name used outside the Go type system. `Version` must be positive.

When an event is published, Letterbox creates an `Envelope` containing a generated message id, event type, event version, sender, occurrence time and serialized event content.

Incoming HTTP messages are expected to use the same envelope shape:

```json
{
  "id": "f90b5c0d-a1b4-4a63-a99c-d6d641cc709a",
  "type": "hello",
  "version": 1,
  "sender": "example-app",
  "occurredAt": "2026-10-03T07:00:00Z",
  "content": {
    "message": "Hello World!"
  }
}
```

## Outbox Flow

`Outbox.Publish` represents the application's intent to publish an event. It does not publish directly to a transport.

```go
err := outbox.Publish(UserCreated{
    UserID: "123",
    Email:  "john@example.com",
})
```

The call performs three steps:

1. Validates and encodes the event into an envelope.
2. Wraps the envelope in a message with `StatusPending`.
3. Saves the message through `OutboxStore.Save`.

Pending messages can be processed explicitly:

```go
err := outbox.Flush()
```

They can also be processed on a schedule:

```go
outbox.Start()
defer outbox.Stop()
```

After a pending message is successfully published by the configured publisher, the Outbox marks it as `StatusEmitted`. If publishing fails with a permanent publishing error, it marks the message as `StatusError`.

The RabbitMQ publisher sends the full `Envelope` as JSON to the configured exchange and uses `Envelope.Type` as the routing key. It uses mandatory publishing and publisher confirms, so an unroutable message or a negative broker confirmation is treated as a publishing failure.

Connection-level RabbitMQ failures are treated differently from permanent publishing failures. If publishing fails because the connection, channel, return notifications or publisher confirm notifications are closed or because waiting for confirmation times out, the publisher closes its RabbitMQ resources and returns a connection error. The Outbox leaves the message as `StatusPending` and stops the current processing pass. A later `Flush` or scheduled run may retry the message if `OutboxStore.Pending` returns it again.

RabbitMQ resources are re-created on the next publishing attempt when the publisher detects that its connection or channel is no longer usable. This supports an at-least-once delivery model when the storage implementation keeps pending messages durable and returns them for later processing. A message whose broker confirmation was lost may be published again, so consumers should treat `Envelope.Id` as the idempotency key.

Retry policy is controlled by the application through the storage implementation. Letterbox does not store attempt counters, backoff timestamps or retry limits. If an application needs backoff, maximum attempts or delayed retries, it should implement that selection in `OutboxStore.Pending(limit)`.

## Outbox Retention

Outbox retention is optional and is configured with `WithOutboxRetention(age, expression)`. When enabled, Letterbox starts a separate retention worker using the provided five-field cron expression.

On each retention run, Letterbox computes a cutoff time in UTC:

```go
olderThan := time.Now().UTC().Add(-age)
```

It then calls:

```go
deleted, err := storage.DeleteExpired(olderThan)
```

The application storage implementation decides what `expired` means. Letterbox does not enforce which statuses, timestamps or records must be deleted. A storage implementation may choose to delete only emitted messages, keep error messages for diagnostics, protect pending messages or apply any other business rule required by the application.

Retention may run concurrently with publishing operations such as `Save`, `Pending` and `UpdateStatus`. `OutboxStore` implementations should therefore make `DeleteExpired` safe for concurrent use with the rest of the outbox storage methods.

## Inbox Flow

The HTTP consumer accepts incoming envelopes on `POST /events` and stores them through `Inbox.Receive`.

```text
HTTP POST /events -> Inbox.Receive -> InboxStore.Save(StatusReceived)
```

The RabbitMQ consumer reads JSON envelopes from the configured queue and stores them through the same `Inbox.Receive` API.

```text
RabbitMQ delivery -> Inbox.Receive -> InboxStore.Save(StatusReceived) -> ACK
```

`Receive` returns:

| Return value | Meaning |
| --- | --- |
| `true, nil` | The message was stored. |
| `false, nil` | The message was already known by storage. |
| `false, error` | The message could not be accepted. |

The HTTP consumer currently returns `201 Created` for a newly stored message and `200 OK` for a duplicate message.

The RabbitMQ consumer acknowledges transport delivery only after the envelope has been accepted by the Inbox:

| Condition | RabbitMQ action |
| --- | --- |
| Invalid JSON or invalid envelope | Reject without requeue. |
| `InboxStore.Save` error | Nack with requeue. |
| Newly stored message | Ack. |
| Duplicate message | Ack. |

RabbitMQ consumption is started when the Inbox is created. `Start` only starts scheduled processing of messages already stored with `StatusReceived`.

Received messages can be processed explicitly:

```go
err := inbox.Flush()
```

They can also be processed on a schedule:

```go
inbox.Start()
defer inbox.Stop()
```

During processing, the Inbox dispatcher selects a registered handler by envelope type and version, decodes the envelope content into the handler event type and executes the handler. On success the message is marked as `StatusExecuted`. On failure it is marked as `StatusError`.

Retry policy is controlled by the application through the storage implementation. Letterbox does not store attempt counters, backoff timestamps or retry limits. If an application needs backoff, maximum attempts or delayed retries, it should implement that selection in `InboxStore.Received(limit)`.

## Inbox Retention

Inbox retention is optional and is configured with `WithInboxRetention(age, expression)`. When enabled, Letterbox starts a separate retention worker using the provided five-field cron expression.

On each retention run, Letterbox computes a cutoff time in UTC:

```go
olderThan := time.Now().UTC().Add(-age)
```

It then calls:

```go
deleted, err := storage.DeleteExpired(olderThan)
```

The application storage implementation decides what `expired` means. Letterbox does not enforce which statuses, timestamps or records must be deleted. A storage implementation may choose to delete only executed messages, keep error messages for diagnostics, protect received messages or apply any other business rule required by the application.

Retention may run concurrently with receiving and processing operations such as `Save`, `Received` and `UpdateStatus`. `InboxStore` implementations should therefore make `DeleteExpired` safe for concurrent use with the rest of the inbox storage methods.

## Reliability Model

Letterbox provides the core Inbox/Outbox coordination model, while storage and transport adapters define the final reliability guarantees of an application.

The library does not provide exactly-once delivery, built-in persistent storage, distributed locking, dead-letter handling, retry counters, retry backoff or transaction management. Applications that need these behaviors should implement them in their storage layer or surrounding infrastructure.

Retry selection is controlled by `OutboxStore.Pending(limit)` and `InboxStore.Received(limit)`. Retention rules are controlled by `DeleteExpired(olderThan)`. Concurrency control for multiple workers is also the responsibility of the storage implementation.

## Configuration

Outbox and Inbox configuration is provided through options passed to `NewOutbox` and `NewInbox`.

### Outbox Options

| Option | Default | Description |
| --- | --- | --- |
| `WithOutboxCron(expression)` | `* * * * *` | Sets the five-field cron expression used by scheduled Outbox processing. |
| `WithOutboxInterval(interval)` | none | Uses a `time.Duration` interval instead of cron scheduling. |
| `WithOutboxPendingLimit(limit)` | `0` | Limits how many pending messages are loaded per processing run. `0` means no limit. |
| `WithOutboxRetention(age, expression)` | disabled | Enables scheduled Outbox retention. `age` must be a positive `time.Duration`; `expression` must be a five-field cron expression. |
| `WithOutboxConsolePublisher()` | required | Uses the console publisher that logs published envelopes. |
| `WithOutboxRabbitMqPublisher(url, exchange, timeout)` | required | Uses RabbitMQ as the Outbox publisher. It publishes envelopes to the configured exchange using the envelope type as the routing key. |

The RabbitMQ publisher requires an existing exchange. The exchange must have a binding that matches the event type used as the routing key. If no queue can be routed for the published event and RabbitMQ returns the message as unroutable, the Outbox marks the message as `StatusError`. If RabbitMQ connectivity fails while publishing or waiting for confirmation, the message remains `StatusPending` and can be retried by a later processing pass.

Example Outbox retention configuration:

```go
outbox, err := letterbox.NewOutbox(
    "example-app",
    storage,
    letterbox.WithOutboxConsolePublisher(),
    letterbox.WithOutboxRetention(7*24*time.Hour, "0 * * * *"),
)
```

### Inbox Options

| Option | Default | Description |
| --- | --- | --- |
| `WithInboxCron(expression)` | `* * * * *` | Sets the five-field cron expression used by scheduled Inbox processing. |
| `WithInboxInterval(interval)` | none | Uses a `time.Duration` interval instead of cron scheduling. |
| `WithInboxReceivedLimit(limit)` | `0` | Limits how many received messages are loaded per processing run. `0` means no limit. |
| `WithInboxRetention(age, expression)` | disabled | Enables scheduled Inbox retention. `age` must be a positive `time.Duration`; `expression` must be a five-field cron expression. |
| `WithInboxHttpConsumer(mux)` | required | Registers the HTTP consumer on the provided `*http.ServeMux`. |
| `WithInboxRabbitMqConsumer(url, queue)` | required | Uses RabbitMQ as the Inbox consumer. It consumes envelopes from the configured queue. |

Example Inbox retention configuration:

```go
inbox, err := letterbox.NewInbox(
    storage,
    letterbox.WithInboxHttpConsumer(mux),
    letterbox.WithInboxRetention(7*24*time.Hour, "0 * * * *"),
)
```

Cron scheduling uses five-field expressions, for example:

```go
letterbox.WithInboxCron("* * * * *")
```

Interval scheduling uses `time.Duration`, for example:

```go
letterbox.WithInboxInterval(time.Second)
letterbox.WithOutboxInterval(time.Minute)
```

If both cron and interval options are provided, the last option applied determines the active scheduling mode.

Configuration validation rejects empty senders, nil storage, invalid cron expressions, non-positive intervals, negative limits and missing publisher or consumer configuration.

The RabbitMQ Inbox consumer requires an existing queue. It does not declare exchanges, queues or bindings. RabbitMQ topology and routing are configured outside the library, for example through a RabbitMQ `definitions.json` file.

Only one Inbox consumer option is required for a given Inbox instance.

## Storage Contracts

### Outbox Storage

Applications provide Outbox persistence by implementing `letterbox.OutboxStore`:

```go
type OutboxStore interface {
    Save(message Message) error
    UpdateStatus(id string, status Status) error
    Pending(limit int) ([]Message, error)
    DeleteExpired(olderThan time.Time) (int, error)
}
```

The Outbox uses this contract as follows:

| Method | Used For |
| --- | --- |
| `Save` | Stores a newly published message with `StatusPending`. |
| `Pending` | Loads messages selected for processing. |
| `UpdateStatus` | Marks processed messages as `StatusEmitted` or `StatusError`. |
| `DeleteExpired` | Deletes messages that the application considers expired before the provided UTC cutoff time. |

`Pending` is also the extension point for application-specific retry policy. Because Letterbox does not persist retry metadata, the storage implementation decides which pending messages are ready for another processing attempt and how the `limit` is applied.

`DeleteExpired` receives a cutoff time calculated by Letterbox, but the storage implementation defines the deletion rule. For example, it may combine the cutoff with message status, update timestamps, emitted timestamps or other business-specific retention criteria. It should be safe to call while other Outbox operations are running.

### Inbox Storage

Applications provide Inbox persistence by implementing `letterbox.InboxStore`:

```go
type InboxStore interface {
    Save(message Message) (bool, error)
    Received(limit int) ([]Message, error)
    UpdateStatus(key MessageKey, status Status) error
    DeleteExpired(olderThan time.Time) (int, error)
}
```

The Inbox uses this contract as follows:

| Method | Used For |
| --- | --- |
| `Save` | Stores an incoming message with `StatusReceived`. Returning `false, nil` indicates that the message was already known by storage. |
| `Received` | Loads received messages selected for processing. |
| `UpdateStatus` | Marks processed messages as `StatusExecuted` or `StatusError`. |
| `DeleteExpired` | Deletes messages that the application considers expired before the provided UTC cutoff time. |

`Received` is also the extension point for application-specific retry policy. Because Letterbox does not persist retry metadata, the storage implementation decides which received messages are ready for another processing attempt and how the `limit` is applied.

`DeleteExpired` receives a cutoff time calculated by Letterbox, but the storage implementation defines the deletion rule. For example, it may combine the cutoff with message status, update timestamps, executed timestamps or other business-specific retention criteria. It should be safe to call while other Inbox operations are running.

## Examples

The repository contains runnable examples for the current Outbox and Inbox flows.

### Outbox Console

[`examples/outbox_console/main.go`](examples/outbox_console/main.go) defines an event, implements an in-memory `OutboxStore`, publishes a message and flushes the Outbox through the console publisher.

Run it with:

```bash
make outbox_console
```

### Outbox RabbitMQ

[`examples/outbox_rabbitmq/main.go`](examples/outbox_rabbitmq/main.go) defines an event, implements an in-memory `OutboxStore`, publishes a message and flushes the Outbox through the RabbitMQ publisher.

The example includes a RabbitMQ Compose setup:

- [`examples/outbox_rabbitmq/docker-compose.yml`](examples/outbox_rabbitmq/docker-compose.yml) starts RabbitMQ with the management plugin.
- [`examples/outbox_rabbitmq/rabbitmq.conf`](examples/outbox_rabbitmq/rabbitmq.conf) loads definitions on startup.
- [`examples/outbox_rabbitmq/definitions.json`](examples/outbox_rabbitmq/definitions.json) defines the example user, exchange, queue and binding.

The Compose setup expects a Docker network named `letterbox`. Create it once with:

```bash
docker network create -d bridge letterbox
```

Start RabbitMQ with:

```bash
docker compose -f examples/outbox_rabbitmq/docker-compose.yml up -d
```

The example Compose setup exposes RabbitMQ on `5672` and the management UI on `15672`.

Run the example with:

```bash
make outbox_rabbitmq
```

The RabbitMQ definitions configure the `letterbox.events` topic exchange, the `example.inbox` queue and a `#` binding. Since the publisher uses the envelope type as the routing key, RabbitMQ topology controls which application inbox queue receives each event.

To verify the message, open the RabbitMQ Management UI at `http://localhost:15672`, log in with the credentials from [`examples/outbox_rabbitmq/definitions.json`](examples/outbox_rabbitmq/definitions.json), open the `example.inbox` queue and use **Get messages** with requeue enabled.

The project is developed in a devcontainer attached to the `letterbox` Docker network, so the example uses the `rabbitmq` host name in its AMQP URL. If you run the Go example directly on the host machine, change the URL in [`examples/outbox_rabbitmq/main.go`](examples/outbox_rabbitmq/main.go) from `rabbitmq:5672` to `localhost:5672`.

### Inbox HTTP

[`examples/inbox_http/main.go`](examples/inbox_http/main.go) defines an event, implements an in-memory `InboxStore`, registers an HTTP consumer and processes received messages every second.

Run it with:

```bash
make inbox_http
```

Send an example event with:

```bash
./examples/inbox_http/request.sh
```

### Inbox RabbitMQ

[`examples/inbox_rabbitmq/main.go`](examples/inbox_rabbitmq/main.go) defines an event, implements an in-memory `InboxStore`, consumes envelopes from RabbitMQ and processes received messages every second.

The example includes a RabbitMQ Compose setup:

- [`examples/inbox_rabbitmq/docker-compose.yml`](examples/inbox_rabbitmq/docker-compose.yml) starts RabbitMQ with the management plugin.
- [`examples/inbox_rabbitmq/rabbitmq.conf`](examples/inbox_rabbitmq/rabbitmq.conf) loads definitions on startup.
- [`examples/inbox_rabbitmq/definitions.json`](examples/inbox_rabbitmq/definitions.json) defines the example user, topic exchange, inbox queue and binding.

The Compose setup expects a Docker network named `letterbox`. Create it once with:

```bash
docker network create -d bridge letterbox
```

Start RabbitMQ with:

```bash
docker compose -f examples/inbox_rabbitmq/docker-compose.yml up -d
```

Run the example with:

```bash
make inbox_rabbitmq
```

The example consumes from the `example.inbox` queue. To send a message manually, open the RabbitMQ Management UI at `http://localhost:15672`, log in with the credentials from [`examples/inbox_rabbitmq/definitions.json`](examples/inbox_rabbitmq/definitions.json), open the `letterbox.events` exchange and publish a message with routing key `hello` and this payload:

```json
{
  "id": "d12183c8-9941-4a19-9eb2-2bf41f30b793",
  "type": "hello",
  "version": 1,
  "sender": "example-app",
  "occurredAt": "2026-10-06T06:12:32.189463586Z",
  "content": {
    "message": "Hello World!"
  }
}
```

RabbitMQ routes the message to `example.inbox` through the `#` binding. The consumer stores it with `StatusReceived`, acknowledges the RabbitMQ delivery and the scheduled Inbox worker dispatches it to the `HelloEvent` handler.
