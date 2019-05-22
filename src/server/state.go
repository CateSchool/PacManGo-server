package main

// a client, represented by an outgoing channel
// We don't need a separate channel for incoming messages for each client. Incoming messages
// have a field that identifies the user. However, clients are tracked independently of the rest of the state.
type client chan<- []byte

// the primary state object
// All field names must be uppercase so the JSON library can access them.
type state struct {
	PlayerStates []playerState
	GameStatus   gameStatus
}
type powerUps struct {
	CanEatGhost bool
}
type playerState struct {
	UserID            string // typically Google sign in
	Longitude         float64
	Latitude          float64
	Role              string // either "ghost" or "pacman"
	Alive             bool
	PowerUps          powerUps
	ConnectedToServer bool // won't be used until a very long time
}
type gameStatus struct {
	IBeaconUUID string // This UUID is shared across the game. All of our clients will continuously broadcast this UUID.
	Started     bool
	TimeElapsed uint64 // in milliseconds
}
