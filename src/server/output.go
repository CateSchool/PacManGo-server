package main

import (
	"encoding/json"
	"fmt"
	"time"
)

// Regularly update clients with the state.
func regularlyUpdateClients(states <-chan state) {
	s := <-states

	ticker := time.NewTicker(100 * time.Millisecond)
	for {
		fmt.Printf("Sending current state to all clients: \n")
		fmt.Println(s)

		// Write message back
		returnMessage, _ := json.Marshal(s)
		outgoingMessages <- returnMessage

		// Update either every 100 milliseconds or when we get a new state.
		select {
		case <-ticker.C:
		case s = <-states:
		}
	}

}
