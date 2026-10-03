curl \
  -X POST \
  -H 'content-type: application/json' \
  -d '{
  "id": "f90b5c0d-a1b4-4a63-a99c-d6d641cc709a",
  "type": "hello",
  "version": 1,
  "sender": "example-app",
  "occurredAt": "2026-10-03T07:00:00Z",
  "content": {
    "message": "Hello World!"
  }
}' \
  'http://127.0.0.1:8080/events'
