/**
 * State Updates
 */

package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
)

type message struct {
	Action    string // currently, can only be updateOwnLocation
	UserID    string
	Latitude  float64
	Longitude float64
}

// Holds the state and listens for and processes update instructions.
// By passing all update instructions through channels and multiplexing on those channels
// with a single process, we make our state updates *atomic*. Why didn't I split this function up?
// I want to establish that everything here must be part of the same process. If I pass references to the state,
// delegate functions could be accidentally called as new goroutines. If I pass the state itself and wait for
// a return of the new state, this could be much slower. But more importantly, it's easier to keep everything in function
// than to enforce a convention of waiting for delegate functions to return.
//
// Eventually, I want to create a channel for each type of message and take care of the JSON parsing somewhere else to take
// advantage of atomic updates through multiplexing.
func processUpdates() {

	// Create the initial state and emit it through the channel.
	var s = state{
		GameStatus: gameStatus{
			IBeaconUUID: "DD09F8AB-0B4A-4890-870D-21ACAA35277F",
		},
	}
	states <- s

	// Keep track of all connected clients (we're working with an abstraction of clients).
	clients := make(map[client]bool)

	// Continuously listen for and process update instructions from various channels.
	for {
		fmt.Printf("Number of clients: %d\n", len(clients))
		select {
		case raw := <-incomingMessages:
			var msg message
			err := json.Unmarshal(raw, &msg)

			if err != nil {
				fmt.Printf("error: %s\n", err)
				return
			}

			fmt.Printf("action: %s\n", msg.Action)

			if msg.Action != "updateOwnLocation" {
				panic(fmt.Sprintf("The only action available is updateOwnLocation, requested %#v", msg.Action))
			}

			// I didn't feel like creating another variable to keep track of whether the player exists in our state.
			func() {
				// Update the player if it exists in our state.
				for i := range s.PlayerStates {
					playerLocation := &s.PlayerStates[i]
					if playerLocation.UserID == msg.UserID {
						playerLocation.Longitude = msg.Longitude
						playerLocation.Latitude = msg.Latitude
						return
					}
				}

				// Or add a new player if it doesn't exist.
				s.PlayerStates = append(s.PlayerStates, playerState{
					UserID:            msg.UserID,
					Longitude:         msg.Longitude,
					Latitude:          msg.Latitude,
					Role:              [2]string{"ghost", "pacman"}[rand.Intn(2)],
					Alive:             true,
					ConnectedToServer: true,
				})
			}()

			// Emit the updated the state.
			states <- s

		case msg := <-outgoingMessages:
			for cli := range clients {
				cli <- msg
			}

		case cli := <-enteringClients:
			clients[cli] = true
		case cli := <-leavingClients:
			delete(clients, cli)
			close(cli)
		}

		fmt.Printf("current state: \n    ")
		fmt.Println(s)
		fmt.Println()
	}
}
