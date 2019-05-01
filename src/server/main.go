package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

/*************
 * State
 *************/

type powerUps struct {
	canEatGhost bool
}
type playerState struct {
	UserID            string
	Longitude         float64
	Latitude          float64
	role              string
	alive             bool
	powerUps          powerUps
	connectedToServer bool
}
type connectionState struct {
	*websocket.Conn
	UserID string
}

type gameStatus struct {
	isBeaconUUID string
	started      bool
	timeElapsed  uint64
}
type state struct {
	messages         chan rawMessage
	playerStates     []playerState
	connectionStates map[int]connectionState
	gameStatus       gameStatus
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

func (s *state) listenForMessages() {
	// by passing all messages through a channel, we make our state updates atomic
	for raw := range s.messages {
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

		// our connections are stateful; update the connection state
		x := s.connectionStates[raw.connectionID]
		x.UserID = msg.UserID
		s.connectionStates[raw.connectionID] = x

		s.updateOwnLocation(msg)
	}
}

/***************/

func (s *state) updateOwnLocation(msg message) {
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
}
func (s *state) deleteConnectionAndPlayer(key int) {
	UserID := s.connectionStates[key].UserID

	if UserID != "" {
		for i := range s.playerStates {
			if s.playerStates[i].UserID == UserID {
				s.playerStates[i] = s.playerStates[len(s.playerStates)-1]
				s.playerStates = s.playerStates[:len(s.playerStates)-1]
				return
			}
		}
	}
	delete(s.connectionStates, key)
}

func main() {

	var s = state{
		messages:         make(chan rawMessage),
		connectionStates: make(map[int]connectionState),
		gameStatus: gameStatus{
			isBeaconUUID: "DD09F8AB-0B4A-4890-870D-21ACAA35277F",
		},
	}

	go s.listenForMessages()

	// Update client at regular intervals with state
	ticker := time.NewTicker(100 * time.Millisecond)
	go func() {
		for range ticker.C {
			returnMessage, _ := json.Marshal(s)

			// Write message back
			for key, conn := range s.connectionStates {
				if err := conn.WriteMessage(websocket.TextMessage, returnMessage); err != nil {
					s.deleteConnectionAndPlayer(key)
				}
			}
		}
	}()

	// Taking messages from the client
	connectionsCounter := 0
	http.HandleFunc("/sockets", func(w http.ResponseWriter, r *http.Request) {
		conn, _ := upgrader.Upgrade(w, r, nil) // error ignored for sake of simplicity

		i := connectionsCounter
		connectionsCounter++

		s.connectionStates[i] = connectionState{conn, ""}

		for {
			// Read message from browser
			_, msg, err := conn.ReadMessage()
			if err != nil {
				fmt.Printf("error reading message: %s\n", err)
				fmt.Printf("    i=%d, state=%+v\n", i, s)
				s.deleteConnectionAndPlayer(i)
				return
			}
			// fmt.Printf("received: %s\n", string(msg))
			s.messages <- rawMessage{connectionID: i, b: msg}
		}
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "websockets.html")
	})

	http.ListenAndServe("172.17.2.225:8080", nil)
	// http.ListenAndServe(":8080", nil)
}
