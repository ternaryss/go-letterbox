package letterbox

import "errors"

type Inbox struct {
	storage         InboxStore
	dispatcher      *dispatcher
	worker          *inboxWorker
	retentionWorker *inboxRetentionWorker
}

func NewInbox(storage InboxStore, options ...InboxOption) (*Inbox, error) {
	if storage == nil {
		return nil, errors.New("storage is nil")
	}

	config := defaultInboxConfig()

	for _, option := range options {
		if err := option(&config); err != nil {
			return nil, err
		}
	}

	if config.consumer == nil {
		return nil, errors.New("inbox consumer is nil")
	}

	dispatcher := newDispatcher()
	worker, err := newInboxWorker(storage, dispatcher, config)

	if err != nil {
		return nil, err
	}

	retentionWorker, err := newInboxRetentionWorker(storage, config)

	if err != nil {
		return nil, err
	}

	inbox := &Inbox{storage: storage, dispatcher: dispatcher, worker: worker, retentionWorker: retentionWorker}

	if err := config.consumer.register(inbox); err != nil {
		return nil, err
	}

	return inbox, nil
}

func (i *Inbox) Start() {
	i.worker.scheduler.Start()

	if i.retentionWorker.enabled {
		i.retentionWorker.scheduler.Start()
	}
}

func (i *Inbox) Stop() error {
	var err error
	err = errors.Join(err, i.worker.scheduler.Shutdown())

	if i.retentionWorker.enabled {
		err = errors.Join(err, i.retentionWorker.scheduler.Shutdown())
	}

	err = errors.Join(err, i.worker.consumer.close())

	return err
}

func (i *Inbox) Flush() error {
	return i.worker.process()
}

func (i *Inbox) Receive(envelope Envelope) (bool, error) {
	if err := envelope.validate(); err != nil {
		return false, err
	}

	message := newMessage(envelope, StatusReceived)

	return i.storage.Save(message)
}
