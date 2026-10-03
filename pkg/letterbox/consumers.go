package letterbox

import (
	"encoding/json"
	"errors"
	"net/http"
)

type consumer interface {
	register(inbox *Inbox) error
}

type httpConsumer struct {
	mux *http.ServeMux
}

func newHttpConsumer(mux *http.ServeMux) *httpConsumer {
	return &httpConsumer{mux: mux}
}

func (c *httpConsumer) register(inbox *Inbox) error {
	if inbox == nil {
		return errors.New("inbox is nil")
	}

	if c.mux == nil {
		return errors.New("http mux is nil")
	}

	c.mux.HandleFunc("POST /events", func(res http.ResponseWriter, req *http.Request) {
		var envelope Envelope

		if err := json.NewDecoder(req.Body).Decode(&envelope); err != nil {
			http.Error(res, "invalid event envelope", http.StatusBadRequest)
			return
		}

		saved, err := inbox.Receive(envelope)

		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}

		if !saved {
			res.WriteHeader(http.StatusOK)
			return
		}

		res.WriteHeader(http.StatusCreated)
	})

	return nil
}
