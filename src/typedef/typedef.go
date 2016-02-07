package typedef

const N_FLOORS int = 4 // TODO -> Do this dynamically.

// Motor commands
const UP = 1
const STOP = 0
const DOWN = -1

// Temporary struct!
// TODO -> Delete!
type SomeStructToPassOnNetwork struct {
	Message  string
	SenderIp string
}

// Enumerators
const (
	BUTTON_CALL_UP = iota
	BUTTON_CALL_DOWN
	BUTTON_COMMAND
	SENSOR_FLOOR
	INDICATOR_FLOOR
	BUTTON_STOP
	SENSOR_OBST
	INDICATOR_DOOR
)
