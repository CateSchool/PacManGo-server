/**
 * Sets up read and write abstractions for a Websocket connection.
 */

package main

import (
	"fmt"
	"log"

	"github.com/gorilla/websocket"
)

// The entire lifetime of each WebSocket connection is handled by this function.
func handleConn(conn *websocket.Conn) {

	// Entering
	log.Println("new client has joined")
	cli := make(chan []byte)
	enteringClients <- cli

	// Relay messages from a channel to WebSocket output.
	go func(conn *websocket.Conn, ch <-chan []byte) {
		for msg := range ch {
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				// This shouldn't happen often, or at all, because we catch attempts to
				// read from closed connections elsewhere.
				fmt.Println(err)
			}
			fmt.Printf("writing %s to %s\n", msg, conn.RemoteAddr())
		}
	}(conn, cli)

	// Relay incoming messages from the connection to a channel shared by the entire program.
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("error reading message: %s\n", err)
			break
		}
		incomingMessages <- msg
	}

	// Leaving: when we can't read any more messages from the network.
	leavingClients <- cli
	conn.Close()

	// Terminate the current process
}
