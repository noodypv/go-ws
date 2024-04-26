package main

import (
	"log"
	"net/http"

	"github.com/noodypv/go-ws/internal/client"
	"github.com/noodypv/go-ws/internal/hub"
	"github.com/sirupsen/logrus"
)

const (
	GUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
)

var (
	h *hub.Hub
)

func main() {
	http.HandleFunc("/", wsHandler())

	h = hub.New()
	go h.Run()

	if err := http.ListenAndServe(":8888", nil); err != nil {
		log.Fatalf("Starting server error: %v", err)
		return
	}
}

func wsHandler() http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {

		c, err := client.New(res, req)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}

		defer func() {
			h.Unsubscribe <- c
			defer c.Close()
			logrus.Info("Disconnected.")
		}()

		h.Subscribe <- c

		for {
			msg, err := c.ReadMessage()
			if err != nil {
				logrus.Errorf("Receiving message error: %v", err)
				return
			}

			if msg.Opcode == client.ConnectionClose {
				return
			} else {
				logrus.Infof("Received message: %v", string(msg.Payload))
			}

			h.Ingoing <- string(msg.Payload)
		}
	}
}
