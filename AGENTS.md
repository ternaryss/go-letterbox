# Project Goal

Letterbox is a generic Go library for reliable asynchronous event delivery and processing using the Inbox/Outbox pattern. It gives applications a small, strongly typed API for publishing outgoing events and handling incoming events while keeping storage and transport details behind interfaces.

## Priorities

- Keep the core library transport-agnostic and storage-agnostic.
- Preserve strongly typed application events at the public API boundary.
- Treat `Publish` as persistence of publishing intent, not direct transport delivery.
- Separate receiving an incoming envelope from processing it with an application handler.
- Keep the public API small and centered on Outbox, Inbox, Publish, Receive, Flush, Start, Stop and Subscribe.
- Model delivery and processing state explicitly with `Message` and `Status`.

# Mental Model

Letterbox is an event-driven Inbox/Outbox library. Applications publish or handle typed Go events, while the library converts them to an internal `Envelope` for persistence and transport boundaries.

The current implementation has two durable flows:

- Outbox: application event -> `Outbox.Publish` -> `Envelope` -> `Message` with `StatusPending` -> `OutboxStore` -> publisher during `Flush` or scheduled processing.
- Inbox: incoming `Envelope` -> `Inbox.Receive` -> `Message` with `StatusReceived` -> `InboxStore` -> dispatcher -> typed handler during `Flush` or scheduled processing.

Core code lives in one package and uses explicit interfaces for infrastructure. Built-in adapters are intentionally dummy-level: console publishing and HTTP receiving.

## Architectural Rules

- `pkg/letterbox/` - core library package containing the public API, event and envelope model, storage contracts, workers, dispatcher and configuration.
- `examples/` - runnable examples that demonstrate library usage.

## Preferred Patterns

- Keep application-facing events strongly typed and require explicit `Type()` and `Version()` methods.
- Use explicit interfaces for infrastructure boundaries: `OutboxStore`, `InboxStore`, internal publisher and internal consumer.
- Keep encode/decode as internal library concerns; application code should normally publish and handle typed events.
- Validate constructor inputs and options early, returning ordinary Go errors.
- Prefer simple structs, package-level constructor functions and functional options for configuration.
- Keep workers focused on one processing pass; `Start` only schedules repeated execution, while `Flush` performs immediate processing.
- Use `MessageKey{Id, Sender}` for Inbox identity and plain message id for Outbox updates, matching existing store contracts.
- Follow existing Go style: small files in one package, unexported helpers for internal concepts, exported names only for the public API.

## Unwanted Patterns

- Do not add core dependencies on a concrete database, ORM, broker, framework, or application architecture.
- Do not expose transport-specific concerns through application event structs.
- Do not infer external event type names from Go type names.
- Do not make `Publish` call a transport directly; it should persist an outgoing message.
- Do not merge transport acknowledgement with application handler execution in the Inbox flow.
- Do not assume exactly-once delivery.
- Do not add transaction, distributed locking, dead-letter, batching, retention, schema registry, or ordering guarantees unless they are explicitly decided.
- Do not expand public API surface when an existing Outbox, Inbox, store, option, worker, or adapter pattern is sufficient.

## Implementation Rules

- Prefer the smallest possible change.
- Do not perform refactoring unrelated to the task.
- Do not create new abstractions without a clear need.
- Preserve consistency with the existing code style.

# Project Analysis

Before analyzing code:

1. Read `ai/snapshot/index.json`.
2. Find the relevant functionality snapshot.
3. Only then analyze code.

# Implementing New Features

1. Find the most similar existing functionality.
2. Treat it as the pattern.
3. Preserve the existing code organization.

# Constraints

- Do not scan the whole repository unnecessarily.
- Prefer analyzing specific files.
- Token efficiency is a priority.

# Important Project Decisions
