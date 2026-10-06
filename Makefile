all: help

help:
	@echo "Command: make [example]"
	@echo ""
	@echo "Available examples:"
	@echo "  - inbox_http (events consumer over HTTP API)"
	@echo "  - outbox_console (publish event & print it's content)"
	@echo "  - outbox_rabbitmq (publish event to RabbitMQ)"

inbox_http:
	@go run ./examples/inbox_http/main.go

outbox_console:
	@go run ./examples/outbox_console/main.go

outbox_rabbitmq:
	@go run ./examples/outbox_rabbitmq/main.go
