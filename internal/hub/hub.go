package hub

import "github.com/noodypv/go-ws/internal/client"

type Hub struct {
	Clients map[*client.Client]bool

	Ingoing chan string

	Subscribe chan *client.Client

	Unsubscribe chan *client.Client
}

func New() *Hub {
	return &Hub{
		Clients: make(map[*client.Client]bool),

		Ingoing: make(chan string, 10),

		Subscribe: make(chan *client.Client, 10),

		Unsubscribe: make(chan *client.Client, 10),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case c := <-h.Subscribe:

			h.Clients[c] = true

		case c := <-h.Unsubscribe:

			delete(h.Clients, c)

		case msg := <-h.Ingoing:

			for c := range h.Clients {
				c.WriteMessage(msg)
			}

		}
	}
}
