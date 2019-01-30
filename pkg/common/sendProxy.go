package common

import (
	"log"
	"strconv"
	"strings"
)

type SendProxy struct {
	sender Sender
}

func NewSendProxy(sender Sender) *SendProxy {
	return &SendProxy{sender}
}

func (p *SendProxy) Send(message string) {
	frags := strings.Split(message, " ")

	m := []byte{}
	for _, f := range frags {
		i, err := strconv.Atoi(f)
		if err != nil {
			log.Println("warn: failed to send: %v", err)
			return
		}
		m = append(m, byte(i))
	}

	err := p.sender.Send(m)
	if err != nil {
		log.Println("warn: failed to send: %v", err)
	}
}
