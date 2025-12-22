package network

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"echo_trace_server/logic"

	"github.com/gorilla/websocket"
)

var Upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Client struct {
	Hub       *Room
	Conn      *websocket.Conn
	Send      chan []byte
	SessionID string
}

func ServeWs(room *Room, w http.ResponseWriter, r *http.Request) {
	conn, err := Upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	// Simple SessionID generation (MVP)
	sessID := fmt.Sprintf("u_%d", time.Now().UnixNano())

	client := &Client{Hub: room, Conn: conn, Send: make(chan []byte, 256), SessionID: sessID}
	client.Hub.Register <- client

	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()
	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		// Parse Input Generic
		var req map[string]interface{}
		if err := json.Unmarshal(message, &req); err != nil {
			continue
		}

		// Handle Messages (MVP: Only Move)
		if typeCode, ok := req["type"].(float64); ok && int(typeCode) == 2001 {
			if payload, ok := req["payload"].(map[string]interface{}); ok {
				if dirMap, ok := payload["dir"].(map[string]interface{}); ok {
					dir := logic.Vector2{
						X: dirMap["x"].(float64),
						Y: dirMap["y"].(float64),
					}
					// Update Game State directly
					c.Hub.GameState.HandleInput(c.SessionID, dir)
				}
			}
		}
	}
}

func (c *Client) writePump() {
	defer func() {
		c.Conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.Conn.WriteMessage(websocket.TextMessage, message)
		}
	}
}

func (c *Client) SendJSON(v interface{}) {
	b, _ := json.Marshal(v)
	c.Send <- b
}

func toJSON(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}
