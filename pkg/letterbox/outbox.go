package letterbox

import "errors"

type Outbox struct {
	sender  string
	storage OutboxStore
	worker  *outboxWorker
}

func NewOutbox(sender string, storage OutboxStore, options ...OutboxOption) (*Outbox, error) {
	if sender == "" {
		return nil, errors.New("sender is empty")
	}

	if storage == nil {
		return nil, errors.New("storage is nil")
	}

	config := defaultOutboxConfig()

	for _, option := range options {
		option(&config)
	}

	if err := config.validate(); err != nil {
		return nil, err
	}

	worker, err := newOutboxWorker(storage, config)

	if err != nil {
		return nil, err
	}

	return &Outbox{sender: sender, storage: storage, worker: worker}, nil
}

func (o *Outbox) Start() {
	o.worker.scheduler.Start()
}

func (o *Outbox) Stop() error {
	return o.worker.scheduler.Shutdown()
}

func (o *Outbox) Flush() error {
	return o.worker.process()
}

func (o *Outbox) Publish(event Event) error {
	envelope, err := encode(event, o.sender)

	if err != nil {
		return err
	}

	message := newMessage(envelope, StatusPending)

	if err := o.storage.Save(message); err != nil {
		return err
	}

	return nil
}
