package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Declare and initialize all of our global variables. Channels are concurrency-safe!
var (
	enteringClients  = make(chan client)
	leavingClients   = make(chan client)
	incomingMessages = make(chan []byte)
	outgoingMessages = make(chan []byte)

	//     State updates are sent through this channel. This allows for a reactive architecture
	// -a one way flow of information. Reactive architectures are common in single-process
	// programs, such as web applications. Go channels enable us to use a reactive architecture
	// safely in our concurrent program, because only one goroutine can directly read the current "global" state.
	//     This could be potentially slow if the struct is relatively large. If I recall correctly, buffering the channel would allow sending to return
	// without waiting for reading. I don't really understand these mechanics, what is considered too large, or
	// the optimal use of channels.
	states = make(chan state, 1)
)

func main() {

	// Prepend the current file and line number to log messages.
	log.SetFlags(log.Lshortfile)

	// Set up our input and output processes.
	go processUpdates()
	go regularlyUpdateClients(states)

	// Assign our handler function to handle WebSocket connections.
	http.HandleFunc("/sockets", func(w http.ResponseWriter, r *http.Request) {
		conn, _ := upgrader.Upgrade(w, r, nil) // error ignored for sake of simplicity
		handleConn(conn)
	})

	// Set up our web page used to test this.
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "websockets.html")
	})

	// Connect our program to the network!
	serverAddress := ":8080"
	fmt.Printf("Server listening on %s\n", serverAddress)
	err := http.ListenAndServe(serverAddress, nil)
	if err != nil {
		log.Fatal(err)
	}
}
