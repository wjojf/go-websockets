package main

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"log"
)

type ClientList map[*Client]bool

type Client struct {
	connection *websocket.Conn
	manager    *Manager

	egress chan Event
}

func NewClient(conn *websocket.Conn, manager *Manager) *Client {
	return &Client{
		connection: conn,
		manager:    manager,
		egress:     make(chan Event),
	}
}

func (c *Client) readMessages() {

	defer func() {
		c.manager.removeClient(c)
	}()

	for {
		_, payload, err := c.connection.ReadMessage()

		if err != nil {

			if websocket.IsUnexpectedCloseError(
				err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error reading message: %v", err)
			}

			return
		}

		var request Event
		if err := json.Unmarshal(payload, &request); err != nil {
			log.Printf("error unmarshaling event: %v", err)
			continue
		}

		if err := c.manager.routeEvent(request, c); err != nil {
			log.Printf("error routing event: %v", err)
			continue
		}
	}
}

func (c *Client) writeMessages() {

	defer func() {
		c.manager.removeClient(c)
	}()

	for {
		select {
		case message, ok := <-c.egress:
			if !ok {
				if err := c.connection.WriteMessage(
					websocket.CloseMessage,
					nil,
				); err != nil {
					log.Printf("error closing connection: %v", err)
				}
				return
			}

			payload, err := json.Marshal(message)
			if err != nil {
				log.Printf("error marshaling event: %v", err)
				continue
			}

			if err := c.connection.WriteMessage(websocket.TextMessage, payload); err != nil {
				log.Printf("error writing message: %v", err)
				continue
			}
		}
	}
}
