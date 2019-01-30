// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import "sync"

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	mtx *sync.Mutex
	// Registered clients.
	clients map[*Client]bool

	// Inbound messages from the clients.
	broadcast chan []byte

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client

	messageHandler func(string)
}

func NewHub(messageHandler func(string)) *Hub {
	return &Hub{
		mtx:            new(sync.Mutex),
		broadcast:      make(chan []byte),
		register:       make(chan *Client),
		unregister:     make(chan *Client),
		clients:        make(map[*Client]bool),
		messageHandler: messageHandler,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mtx.Lock()

			h.clients[client] = true

			h.mtx.Unlock()
		case client := <-h.unregister:
			h.mtx.Lock()

			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}

			h.mtx.Unlock()
		case message := <-h.broadcast:
			h.mtx.Lock()

			h.messageHandler(string(message))

			h.mtx.Unlock()
		}
	}
}

func (h *Hub) Broadcast(message string) error {
	h.mtx.Lock()
	defer h.mtx.Unlock()

	for client := range h.clients {
		select {
		case client.send <- []byte(message):
		default:
			close(client.send)
			delete(h.clients, client)
		}
	}

	return nil
}
