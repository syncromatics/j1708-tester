package simma

import (
	"context"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/pkg/errors"
)

type protocol struct {
	writeMtx    *sync.Mutex
	writeBuffer []byte

	channel      *channel
	acks         chan *ack
	j1587Handler func(*j1587Message)
	badBytes     int
}

func newProtocol(port string, j1587Handler func(*j1587Message)) (*protocol, error) {
	p := &protocol{
		writeMtx:     new(sync.Mutex),
		writeBuffer:  make([]byte, 2000),
		acks:         make(chan *ack),
		j1587Handler: j1587Handler,
	}

	c, err := openChannel(port, p.parseMessage)
	if err != nil {
		return nil, errors.Wrap(err, "failed opening channel")
	}

	p.channel = c

	return p, nil
}

func (p *protocol) Start(ctx context.Context) func() error {
	return func() error {
		for {
			select {
			case <-ctx.Done():
				return nil
			}
		}
	}
}

func (p *protocol) Send(message io.Writer) error {
	p.writeMtx.Lock()
	defer p.writeMtx.Unlock()

	l, err := message.Write(p.writeBuffer)
	if err != nil {
		return errors.Wrap(err, "failed to write to out buffer")
	}

	for i := 0; i < 3; i++ {
		err = p.channel.write(p.writeBuffer[:l])
		if err != nil {
			return errors.Wrap(err, "failed writing to channel")
		}

		timer := time.NewTimer(3 * time.Second)

		loop := true
		for loop {
			select {
			case ack := <-p.acks:
				if ack.MessageIdentifier == int(p.writeBuffer[0]) {
					return nil
				}
				break
			case <-timer.C:
				loop = false
			}
		}
	}

	return fmt.Errorf("failed to receive ack after 3 retries")
}

func (p *protocol) parseMessage(message []byte) {
	switch message[0] {
	case 0:
		ack, err := newAck(message)
		if err != nil {
			log.Printf("warn: %v", err)
			return
		}

		p.acks <- ack
		break
	case 22:
		m, err := newJ1587Message(message)
		if err != nil {
			log.Printf("warn: %v", err)
			return
		}

		p.j1587Handler(m)

		break
	case 23: // stats
		stats, err := newStats(message)
		if err != nil {
			log.Printf("warn: %v\n", err)
			return
		}

		if stats.TotalInvalidJ1708Bytes > 0 && stats.TotalInvalidJ1708Bytes != p.badBytes {
			log.Printf("warn: invalid j1708 bytes received %d", stats.TotalInvalidJ1708Bytes)
			p.badBytes = stats.TotalInvalidJ1708Bytes
		}
		break
	default:
		break
	}
}
