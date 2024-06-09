package gows

import (
	"bufio"
	"encoding/binary"
	"io"
	"net"
)

type opcode byte

const (
	Continue opcode = iota
	Text
	Binary
	Close opcode = iota + 5
	Ping
	Pong
)

type Message struct {
	IsFinal       bool
	OperationCode opcode
	IsMasking     bool
	Payload       []byte
}

type Client struct {
	conn  net.Conn
	bufrw *bufio.ReadWriter
}

func (c *Client) Send(msg *Message) error {
	buf := make([]byte, 2)
	buf[0] |= byte(msg.OperationCode)

	buf[0] |= 0x80

	size := len(msg.Payload)

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

	buf = append(buf, msg.Payload...)

	_, err := c.bufrw.Write(buf)
	if err != nil {
		return err
	}

	if err := c.bufrw.Flush(); err != nil {
		return err
	}

	return nil
}

func (c *Client) Read() (*Message, error) {
	msg := Message{}

	buf := make([]byte, 2, 12)

	_, err := c.bufrw.Read(buf)
	if err != nil {
		return nil, err
	}

	if buf[0]>>7 == 1 {
		msg.IsFinal = true
	}

	msg.OperationCode = opcode(buf[0] & 0xf)

	if buf[1]>>7 == 1 {
		msg.IsMasking = true
	}

	rest := 0
	if msg.IsMasking {
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

		_, err = c.bufrw.Read(buf)
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
	if msg.IsMasking {
		mask = buf
	}

	msg.Payload = make([]byte, size)

	_, err = io.ReadFull(c.bufrw, msg.Payload)
	if err != nil {
		return nil, err
	}

	if msg.IsMasking {
		for i := 0; i < len(msg.Payload); i++ {
			msg.Payload[i] ^= mask[i%4]
		}
	}

	return &msg, nil
}

func (c *Client) Close() error {
	buf := make([]byte, 2)
	buf[0] |= byte(Close)

	buf[0] |= 0x80

	if _, err := c.bufrw.Write(buf); err != nil {
		return err
	}

	if err := c.bufrw.Flush(); err != nil {
		return err
	}

	return c.conn.Close()
}
