package simma

import (
	"fmt"
	"io"
	"log"
	"sync"

	"github.com/jacobsa/go-serial/serial"
	"github.com/pkg/errors"
)

type channel struct {
	portName string
	port     io.ReadWriteCloser

	writeMtx    *sync.Mutex
	writeBuffer []byte
}

func openChannel(portName string, receiver func([]byte)) (*channel, error) {
	channel := &channel{
		writeMtx:    new(sync.Mutex),
		writeBuffer: make([]byte, 2000),
		portName:    portName,
	}

	options := serial.OpenOptions{
		PortName:        portName,
		BaudRate:        115200,
		DataBits:        8,
		StopBits:        1,
		MinimumReadSize: 4,
	}

	port, err := serial.Open(options)
	if err != nil {
		return nil, errors.Wrapf(err, "failed opening port '%s'", portName)
	}
	channel.port = port

	go channel.connect(receiver)

	return channel, nil
}

func (c *channel) connect(receiver func([]byte)) {
	messageBuffer := make([]byte, 1024)
	lb := make([]int, 2)
	li := 0

	s := 0
	e := false

	i := 0
	length := 0
	buffer := make([]byte, 1024)

	for {
		r, err := c.port.Read(buffer)
		if err != nil {
			log.Fatal(errors.Wrapf(err, "failed reading from port %s", c.portName))
		}

		for a := 0; a < r; a++ {
			b := buffer[a]

			if b == 192 { // start byte always resets
				s = 1
				li = 0
				continue
			}

			if b == 219 { // escaping
				e = true
				continue
			}

			if e {
				e = false

				switch b {
				case 220:
					b = 192
					break
				case 221:
					b = 219
					break
				default: // bad escape
					s = 0
					continue
				}
			}

			switch s {
			case 0: // no start
				break
			case 1: // getting length
				lb[li] = int(b)
				li++
				if li == 2 {
					length = int(lb[0]<<8+lb[1]) - 1
					s = 2
					i = 0
				}
				break

			case 2: // getting message
				messageBuffer[i] = b
				i++
				if i == length {
					s = 3
				}
				break

			case 3: //checksum
				s = 0

				cs := 0
				cs = (lb[0] + cs) & 0xFF
				cs = (lb[1] + cs) & 0xFF

				for j := 0; j < length; j++ {
					cs = (int(messageBuffer[j]) + cs) & 0xFF
				}

				cs = 256 - cs

				if cs == int(b) {
					receiver(messageBuffer[:length])
				} else {
					log.Printf("warn checksum failed")
				}

				break
			}
		}
	}
}

func (c *channel) write(message []byte) error {
	c.writeMtx.Lock()
	defer c.writeMtx.Unlock()

	l := len(message) + 1

	c.writeBuffer[0] = 192

	i := 1
	writebyte := func(b byte) {
		switch b {
		case 192:
			c.writeBuffer[i] = 219
			i++
			c.writeBuffer[i] = 220
			i++
			break

		case 219:
			c.writeBuffer[i] = 219
			i++
			c.writeBuffer[i] = 221
			i++
			break

		default:
			c.writeBuffer[i] = b
			i++
			break
		}
	}

	writebyte(byte(l >> 8))
	writebyte(byte(l))

	for _, b := range message {
		writebyte(b)
	}

	cs := 0
	cs = (l>>8 + cs) & 0xFF
	cs = (l + cs) & 0xFF

	for j := 0; j < len(message); j++ {
		cs = (int(message[j]) + cs) & 0xFF
	}

	cs = 256 - cs

	writebyte(byte(cs))

	m := c.writeBuffer[:i]

	r, err := c.port.Write(m)
	if r != i {
		return fmt.Errorf("failed")
	}
	if err != nil {
		return errors.Wrap(err, "failed writing message to port")
	}

	return nil
}
