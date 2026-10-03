package letterbox

import "errors"

type Inbox struct {
	storage    InboxStore
	dispatcher *dispatcher
	worker     *inboxWorker
}

func NewInbox(storage InboxStore, options ...InboxOption) (*Inbox, error) {
	if storage == nil {
		return nil, errors.New("storage is nil")
	}

	config := defaultInboxConfig()

	for _, option := range options {
		option(&config)
	}

	if err := config.validate(); err != nil {
		return nil, err
	}

	dispatcher := newDispatcher()
	worker, err := newInboxWorker(storage, dispatcher, config)

	if err != nil {
		return nil, err
	}

	inbox := &Inbox{storage: storage, dispatcher: dispatcher, worker: worker}

	if err := config.consumer.register(inbox); err != nil {
		return nil, err
	}

	return inbox, nil
}

func (i *Inbox) Start() {
	i.worker.scheduler.Start()
}

func (i *Inbox) Stop() error {
	return i.worker.scheduler.Shutdown()
}

func (i *Inbox) Flush() error {
	return i.worker.process()
}

func (i *Inbox) Receive(envelope Envelope) (bool, error) {
	if envelope.Id == "" {
		return false, errors.New("envelope id is empty")
	}

	if envelope.Type == "" {
		return false, errors.New("envelope type is empty")
	}

	if envelope.Version <= 0 {
		return false, errors.New("envelope version must be positive")
	}

	if envelope.Sender == "" {
		return false, errors.New("envelope sender is empty")
	}

	message := newMessage(envelope, StatusReceived)

	return i.storage.Save(message)
}
