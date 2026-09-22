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
