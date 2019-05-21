package main

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

type gameStatus struct {
	iBeaconUUID string // This UUID is shared across the game. All of our clients will continuously broadcast this UUID.
	started     bool
	timeElapsed uint64
}
type state struct {
	playerStates []playerState
	gameStatus   gameStatus
}
