package simma

import (
	"context"

	"github.com/pkg/errors"
	"github.com/syncromatics/j1708-tester/pkg/common"
	"golang.org/x/sync/errgroup"
)

type Device struct {
	port string

	j1587Handler func(*common.J1587Message)
	protocol     *protocol
}

func NewDevice(port string, j1587Handler func(*common.J1587Message)) *Device {
	return &Device{
		port:         port,
		j1587Handler: j1587Handler,
	}
}

func (d *Device) Open(ctx context.Context) func() error {
	cc, cancel := context.WithCancel(ctx)
	grp, cc := errgroup.WithContext(cc)

	return func() error {
		p, err := newProtocol(d.port, d.handleJ1587)
		if err != nil {
			return errors.Wrap(err, "Failed creating protocol")
		}
		d.protocol = p

		grp.Go(p.Start(cc))

		err = p.Send(passAllModeConfig{
			port:  0,
			j1587: false,
			j1708: true,
			j1939: false,
			can:   false,
		})
		if err != nil {
			return errors.Wrap(err, "failed to enabled pass all mode")
		}

		select {
		case <-ctx.Done():
		case <-cc.Done():
			break
		}

		cancel()
		return grp.Wait()
	}
}

func (d *Device) Send(message []byte) error {
	mid := int(message[0])
	pid := int(message[1])

	err := d.protocol.Send(&j1587Message{
		Mid:  mid,
		Pid:  pid,
		Data: message[2:],
	})
	if err != nil {
		return errors.Wrap(err, "failed to send j1587 message")
	}
	return nil
}

func (d *Device) handleJ1587(m *j1587Message) {
	d.j1587Handler(&common.J1587Message{
		Mid:  m.Mid,
		Pid:  m.Pid,
		Data: m.Data,
		Raw:  m.Raw,
	})
}
