# go-ws [In progress...]
A simple implementation of WebSocket Protocol according to RFC6455. 

# Usage example
```go
package main

import (
	"log"
	"net/http"

	"github.com/noodypv/go-ws/pkg/gows"
)

func main() {
	http.HandleFunc("/ws", wsHandler)

	if err := http.ListenAndServe(":8999", nil); err != nil {
		log.Fatal(err)
	}
}

func wsHandler(res http.ResponseWriter, req *http.Request) {
	c, err := gows.Accept(res, req)
	if err != nil {
		log.Println(err)
		return
	}

	c.Send(&gows.Message{
		IsFinal:       true,
		OperationCode: gows.Text,
		IsMasking:     true,
		Payload:       []byte("Hellow bros."),
	})

	go func(cl *gows.Client) {
		defer cl.Close()

		for {
			msg, err := cl.Read()
			if err != nil {
				log.Println(err)
				return
			}

			log.Println(string(msg.Payload))
		}
	}(c)
}

```
