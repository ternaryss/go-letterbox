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
	return errors.Join(i.worker.scheduler.Shutdown(), i.worker.consumer.close())
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
