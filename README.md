# Letterbox

Letterbox is a generic Go library for reliable asynchronous event delivery and processing using the Inbox/Outbox pattern.

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

Once an event is persisted, Letterbox takes responsibility for its further delivery or processing, including failure handling and retries.

Applications operate on strongly typed events, while Letterbox uses a generic event envelope internally to exchange and persist them.

## Goals

- Transport-agnostic event processing
- Storage-agnostic Inbox and Outbox
- Reliable event delivery and processing
- Explicit delivery and processing state
- Retry and duplicate handling
- Strongly typed application events
- Minimal integration with application code

## Outbox API

The Outbox API is exposed from the `github.com/ternaryss/go-letterbox/pkg/letterbox` package.

```go
outbox, err := letterbox.NewOutbox(sender, storage, options...)
```

`NewOutbox` requires:

- `sender`: a non-empty identifier of the publishing application.
- `storage`: an implementation of `letterbox.OutboxStore`.
- `options`: optional `letterbox.OutboxOption` values.

Available Outbox methods:

| Method | Description |
| --- | --- |
| `Publish(event Event) error` | Encodes an application event and stores it as a pending message. |
| `Flush() error` | Processes pending messages once. |
| `Start()` | Starts scheduled processing of pending messages. |
| `Stop() error` | Stops the scheduler used by `Start`. |

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

The built-in publisher currently available through public options is the console publisher configured by `WithConsolePublisher`. It logs the envelope with `log/slog`.

## Outbox Configuration

Outbox configuration is provided through options passed to `NewOutbox`.

| Option | Default | Description |
| --- | --- | --- |
| `WithCron(expression)` | `* * * * *` | Sets the five-field cron expression used by scheduled processing. |
| `WithPendingLimit(limit)` | `0` | Limits how many pending messages are loaded per processing run. `0` means no limit. |
| `WithConsolePublisher()` | enabled by default | Uses the console publisher that logs published envelopes. |

Configuration validation rejects empty senders, nil storage, invalid cron expressions, negative pending limits and nil publisher configuration.

## Outbox Storage Contract

Applications provide Outbox persistence by implementing `letterbox.OutboxStore`:

```go
type OutboxStore interface {
    Save(message Message) error
    UpdateStatus(id string, status Status) error
    Pending(limit int) ([]Message, error)
}
```

The Outbox uses this contract as follows:

| Method | Used For |
| --- | --- |
| `Save` | Stores a newly published message with `StatusPending`. |
| `Pending` | Loads messages selected for processing. |
| `UpdateStatus` | Marks processed messages as `StatusEmitted` or `StatusError`. |

Storage implementation details are left to the application or adapter using the library.

## Example Usage

The repository contains a runnable Outbox example in [`examples/outbox_console/main.go`](examples/outbox_console/main.go). It defines an event, implements an in-memory `OutboxStore`, publishes a message and flushes the Outbox through the console publisher.

Run it with:

```bash
make outbox_console
```
