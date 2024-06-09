package gows

import (
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"net/http"
)

var (
	errInvalidConnUpgadeRequest = errors.New("invalid connection upgrade request. WS upgrade failed")
	errEmptySecWSKey            = errors.New("empty Sec-Websocket-Key. WS upgrade failed")
	errHijack                   = errors.New("cannot take over the connection. WS upgrade failed")
)

const (
	guid = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
)

func Accept(res http.ResponseWriter, req *http.Request) (*Client, error) {
	h := req.Header.Get("Connection")
	if h != "Upgrade" {
		return nil, errInvalidConnUpgadeRequest
	}

	h = req.Header.Get("Upgrade")
	if h != "websocket" {
		return nil, errInvalidConnUpgadeRequest
	}

	key := req.Header.Get("Sec-Websocket-Key")

	if key == "" {
		return nil, errEmptySecWSKey
	}

	hash := sha1.Sum([]byte(key + guid))
	str := base64.StdEncoding.EncodeToString(hash[:])

	hj, ok := res.(http.Hijacker)
	if !ok {
		return nil, errHijack
	}

	conn, bufrw, err := hj.Hijack()
	if err != nil {
		return nil, errHijack
	}

	bufrw.WriteString("HTTP/1.1 101 Switching Protocols\r\n")
	bufrw.WriteString("Connection: Upgrade\r\n")
	bufrw.WriteString("Upgrade: websocket\r\n")
	bufrw.WriteString("Sec-Websocket-Accept: " + str + "\r\n\r\n")
	bufrw.Flush()

	return &Client{
		conn:  conn,
		bufrw: bufrw,
	}, nil
}
