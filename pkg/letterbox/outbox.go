package letterbox

import "errors"

type Outbox struct {
	sender          string
	storage         OutboxStore
	worker          *outboxWorker
	retentionWorker *outboxRetentionWorker
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
		if err := option(&config); err != nil {
			return nil, err
		}
	}

	if config.publisher == nil {
		return nil, errors.New("outbox publisher is nil")
	}

	worker, err := newOutboxWorker(storage, config)

	if err != nil {
		return nil, err
	}

	retentionWorker, err := newOutboxRetentionWorker(storage, config)

	if err != nil {
		return nil, err
	}

	return &Outbox{sender: sender, storage: storage, worker: worker, retentionWorker: retentionWorker}, nil
}

func (o *Outbox) Start() {
	o.worker.scheduler.Start()

	if o.retentionWorker.enabled {
		o.retentionWorker.scheduler.Start()
	}
}

func (o *Outbox) Stop() error {
	var err error
	err = errors.Join(err, o.worker.scheduler.Shutdown())

	if o.retentionWorker.enabled {
		err = errors.Join(err, o.retentionWorker.scheduler.Shutdown())
	}

	err = errors.Join(err, o.worker.publisher.close())

	return err
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
