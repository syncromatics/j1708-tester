package simma

import (
	"encoding/binary"
	"fmt"
)

type passAllModeConfig struct {
	port  byte
	j1708 bool
	j1587 bool
	can   bool
	j1939 bool
}

func (m passAllModeConfig) Write(p []byte) (int, error) {
	if len(p) < 6 {
		return 0, fmt.Errorf("byte slice length '%d' expected at least '6'", len(p))
	}

	p[0] = 18
	p[1] = m.port

	if m.j1708 {
		p[2] = 1
	} else {
		p[2] = 0
	}

	if m.j1587 {
		p[3] = 1
	} else {
		p[3] = 0
	}

	if m.can {
		p[4] = 1
	} else {
		p[4] = 0
	}

	if m.j1939 {
		p[5] = 1
	} else {
		p[5] = 0
	}

	return 6, nil
}

type stats struct {
	TotalValidJ1708Messages int
	TotalInvalidJ1708Bytes  int
	TotalCANFrames          int
	HardwareVersion         int
	SoftwareVersion         int
}

func newStats(m []byte) (*stats, error) {
	if len(m) != 15 {
		return nil, fmt.Errorf("stats message should be '11' got '%d'", len(m))
	}

	stats := stats{}

	stats.TotalValidJ1708Messages = int(binary.BigEndian.Uint32(m[1:]))
	stats.TotalInvalidJ1708Bytes = int(binary.BigEndian.Uint32(m[5:]))
	stats.TotalCANFrames = int(binary.BigEndian.Uint32(m[9:]))

	stats.HardwareVersion = int(m[13])
	stats.SoftwareVersion = int(m[14])

	return &stats, nil
}

type ack struct {
	MessageIdentifier int
}

func newAck(message []byte) (*ack, error) {
	if len(message) != 2 {
		return nil, fmt.Errorf("ack message should be '2' got '%d'", len(message))
	}

	return &ack{int(message[1])}, nil
}

type j1587Message struct {
	Mid  int
	Pid  int
	Data []byte
	Raw  []byte
}

func newJ1587Message(message []byte) (*j1587Message, error) {
	if len(message) < 3 {
		return nil, fmt.Errorf("failed parsing j1587 message expected length > '2' got '%d'", len(message))
	}

	m := &j1587Message{}
	m.Raw = []byte{}

	m.Mid = int(message[1])
	m.Raw = append(m.Raw, message[1])

	m.Raw = append(m.Raw, message[2])
	if message[2] == 255 {
		m.Raw = append(m.Raw, message[3])
		m.Data = message[4:]
		m.Pid = 256 + int(message[4])
	} else {
		m.Pid = int(message[2])
		m.Data = message[3:]
	}

	for _, b := range m.Data {
		m.Raw = append(m.Raw, b)
	}

	return m, nil
}

func (m *j1587Message) Write(p []byte) (int, error) {
	r := 5 + len(m.Data)
	if len(p) < r {
		return 0, fmt.Errorf("byte slice length '%d' expected at least '6'", r)
	}

	p[0] = 8
	p[1] = byte(m.Mid)
	p[2] = byte(m.Pid >> 8)
	p[3] = byte(m.Pid)
	p[4] = 4

	i := 5
	for _, b := range m.Data {
		p[i] = b
		i++
	}

	return i, nil
}
