package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

/****************
 * Message types
 ****************/

type rawMessage struct {
	b            []byte
	connectionID int
}
type message struct {
	Action    string
	UserID    string
	Latitude  float64
	Longitude float64
}

/*****************
 * State Updates
 *****************/

type client chan<- []byte // data to write to the client

var (
	enteringClients  = make(chan client)
	leavingClients   = make(chan client)
	incomingMessages = make(chan rawMessage)
	outgoingMessages = make(chan []byte)
	states           = make(chan state)
)

func handleConn(conn *websocket.Conn) {
	log.Println("new client has joined")
	cli := make(chan []byte)
	enteringClients <- cli

	go writeClient(conn, cli)

	// Read message from browser
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("error reading message: %s\n", err)
			// fmt.Printf("    i=%d, state=%+v\n", i, s)
			break
		}
		incomingMessages <- rawMessage{b: msg}
	}

	// Leaving
	leavingClients <- cli
	conn.Close()
}

// by passing all messages through a channel, we make our state updates atomic
func listenForMessages() {

	var s = state{
		gameStatus: gameStatus{
			iBeaconUUID: "DD09F8AB-0B4A-4890-870D-21ACAA35277F",
		},
	}
	states <- s
	clients := make(map[client]bool) // all connected clients

	for {
		fmt.Printf("Number of clients: %d\n", len(clients))
		select {
		case raw := <-incomingMessages:
			var msg message
			err := json.Unmarshal(raw.b, &msg)

			if err != nil {
				fmt.Printf("error: %s\n", err)
				return
			}

			fmt.Printf("action: %s\n", msg.Action)

			if msg.Action != "updateOwnLocation" {
				panic(fmt.Sprintf("The only action available is updateOwnLocation, requested %#v", msg.Action))
			}

			for i := range s.playerStates {
				playerLocation := &s.playerStates[i]
				if playerLocation.UserID == msg.UserID {
					playerLocation.Longitude = msg.Longitude
					playerLocation.Latitude = msg.Latitude
					return
				}
			}

			// add a new Player
			s.playerStates = append(s.playerStates, playerState{
				UserID:            msg.UserID,
				Longitude:         msg.Longitude,
				Latitude:          msg.Latitude,
				role:              [2]string{"ghost", "pacman"}[rand.Intn(2)],
				alive:             true,
				connectedToServer: true,
			})
			fmt.Printf("adding new player, new playerStates: \n %+v", s.playerStates)

			//go func(s state) {
			states <- s
			// }(s)
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

		fmt.Printf("current state: \n")
		fmt.Println(s)
		fmt.Println()
	}
}

// Regularly update clients with the state
func regularlyUpdateClients(states <-chan state) {
	s := <-states

	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		for {
			fmt.Printf("Sending current state to all clients: \n")
			fmt.Println(s)

			// Write message back
			returnMessage, _ := json.Marshal(s)
			outgoingMessages <- returnMessage

			// update either every 100 milliseconds or when we get a new state
			select {
			case <-ticker.C:
			case s = <-states:
			}
		}
	}()
}

func writeClient(conn *websocket.Conn, ch <-chan []byte) {
	for msg := range ch {
		if err := conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
			// delete the client if we don't need him anymore
			// s.deleteConnectionAndPlayer(key)
		}
	}
}

/***********/

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	go listenForMessages()
	go regularlyUpdateClients(states)

	http.HandleFunc("/sockets", func(w http.ResponseWriter, r *http.Request) {
		log.Println("new client has joined")
		conn, _ := upgrader.Upgrade(w, r, nil) // error ignored for sake of simplicity
		handleConn(conn)
	})

	// serverAddress := "172.17.2.225:8080"
	serverAddress := ":8000"

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "websockets.html")
	})

	fmt.Printf("Server listening on %s\n", serverAddress)
	err := http.ListenAndServe(serverAddress, nil)
	if err != nil {
		log.Fatal(err)
	}
}
