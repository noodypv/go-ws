package client

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"net/http"

	"github.com/sirupsen/logrus"
)

type opcode byte

const (
	Continuation opcode = iota
	Text
	Binary
	ConnectionClose opcode = iota + 5
	Ping
	Pong
)

type Client struct {
	conn net.Conn

	bufrw *bufio.ReadWriter
}

type Message struct {
	FinBit  bool
	Opcode  opcode
	maskBit bool
	Payload []byte
}

var (
	ErrInvalidConnUpgateRequest = errors.New("invalid connection upgrade request. WS upgrade failed")
	ErrEmptySecWSKey            = errors.New("empty Sec-Websocket-Key. WS upgrade failed")
	ErrHijack                   = errors.New("cannot take over the connection. WS upgrade failed")
)

const (
	GUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
)

func New(res http.ResponseWriter, req *http.Request) (*Client, error) {
	h := req.Header.Get("Connection")
	if h != "Upgrade" {
		return nil, ErrInvalidConnUpgateRequest
	}

	h = req.Header.Get("Upgrade")
	if h != "websocket" {
		return nil, ErrInvalidConnUpgateRequest
	}

	key := req.Header.Get("Sec-Websocket-Key")

	if key == "" {
		return nil, ErrEmptySecWSKey
	}

	hash := sha1.Sum([]byte(key + GUID))
	str := base64.StdEncoding.EncodeToString(hash[:])

	hj, ok := res.(http.Hijacker)
	if !ok {
		return nil, ErrHijack
	}

	conn, bufrw, err := hj.Hijack()
	if err != nil {
		return nil, ErrHijack
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

func (c *Client) WriteMessage(msg string) {
	buf := make([]byte, 2)
	buf[0] |= byte(Text)

	buf[0] |= 0x80

	size := len([]byte(msg))

	if size < 126 {
		buf[1] |= byte(size)
	} else if size < 1<<16 {
		buf[1] |= 126

		sendsize := make([]byte, 2)
		binary.BigEndian.PutUint16(sendsize, uint16(size))
		buf = append(buf, sendsize...)
	} else {
		buf[1] |= 127

		sendsize := make([]byte, 8)
		binary.BigEndian.PutUint64(sendsize, uint64(size))
		buf = append(buf, sendsize...)
	}

	buf = append(buf, []byte(msg)...)

	c.bufrw.Write(buf)
	c.bufrw.Flush()
}

func (c *Client) ReadMessage() (*Message, error) {
	f := Message{}

	buf := make([]byte, 2, 12)

	_, err := c.bufrw.Read(buf)
	if err != nil {
		return nil, err
	}

	if buf[0]>>7 == 1 {
		f.FinBit = true
	}

	f.Opcode = opcode(buf[0] & 0xf)

	if buf[1]>>7 == 1 {
		f.maskBit = true
	}

	rest := 0
	if f.maskBit {
		rest += 4
	}

	size := uint64(buf[1] & 0x7f)

	switch size {
	case 126:
		rest += 2
	case 127:
		rest += 8
	}

	if rest > 0 {
		buf = buf[:rest]

		_, err := c.bufrw.Read(buf)
		if err != nil {
			return nil, err
		}

		switch size {
		case 126:
			size = uint64(binary.BigEndian.Uint16(buf[:2]))
			buf = buf[2:]
		case 127:
			size = uint64(binary.BigEndian.Uint64(buf[:8]))
			buf = buf[8:]
		}
	}

	var mask []byte
	if f.maskBit {
		mask = buf
	}

	f.Payload = make([]byte, size)

	_, err = io.ReadFull(c.bufrw, f.Payload)
	if err != nil {
		return nil, err
	}

	if f.maskBit {
		for i := 0; i < len(f.Payload); i++ {
			f.Payload[i] ^= mask[i%4]
		}
	}

	return &f, nil
}

func (c *Client) Close() {
	buf := make([]byte, 2)
	buf[0] |= byte(ConnectionClose)

	buf[0] |= 0x80

	c.bufrw.Write(buf)
	c.bufrw.Flush()

	logrus.Info("Connection closed.")

	c.conn.Close()
}
