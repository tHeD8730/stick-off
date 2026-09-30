package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"nhooyr.io/websocket"
)

type Client struct {
	hub       *Hub
	conn      *websocket.Conn
	send      chan []byte
	roomCode  string
	playerIdx int
	name      string
}

func serveWS(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		log.Printf("ws accept: %v", err)
		return
	}

	c := &Client{
		hub:  hub,
		conn: conn,
		send: make(chan []byte, 64),
	}

	hub.register <- c

	go c.writePump()
	c.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close(websocket.StatusNormalClosure, "")
	}()

	ctx := context.Background()
	for {
		_, data, err := c.conn.Read(ctx)
		if err != nil {
			break
		}
		c.hub.incoming <- clientMsg{client: c, data: data}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	ctx := context.Background()
	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				return
			}
			if err := c.conn.Write(ctx, websocket.MessageText, msg); err != nil {
				return
			}
		case <-ticker.C:
			// ping
			if err := c.conn.Ping(ctx); err != nil {
				return
			}
		}
	}
}

func (c *Client) sendError(msg string) {
	d, _ := json.Marshal(OutMessage{Type: MsgError, Message: msg, PlayerIdx: c.playerIdx})
	select {
	case c.send <- d:
	default:
	}
}
