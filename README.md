# go-ws
A simple implementation of WebSocket Protocol according to RFC6455. 

# approach
On every WebSocket upgrade connection creates client that stores in a hub's map. When there's a new message from a client, hub receives it and echoes to other clients iterating over internal map of clients.
