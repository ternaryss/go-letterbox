all: help

help:
	@echo "Command: make [example]"
	@echo ""
	@echo "Available examples:"
	@echo "  - outbox_console (publish event & print it's content)"

outbox_console:
	@go run ./examples/outbox_console/main.go
