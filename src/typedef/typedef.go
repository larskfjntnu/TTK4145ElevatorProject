package typedef

const N_FLOORS int = 4 // TODO -> Do this dynamically.
const N_BUTTONS int = 3

// --------------- EVENT STRUCTS -----------------
type ButtonEvent struct{
	ButtonType int
	Floor int
}

type LightEvent struct{
	LightType int
	Floor int
	Value bool
}

type MotorEvent struct{
	MotorDirection Direction
}

type FloorEvent struct{
	CurrentDirection Direction
	Floor int
}


// --------------------- "Enumerators" --------------------

type Direction int 
const(
	DIR_DOWN Direction = -1 << iota 
	DIR_STOP
	DIR_UP
)

// Enumerators
const (
	BUTTON_CALL_UP = iota
	BUTTON_CALL_DOWN
	BUTTON_COMMAND
	SENSOR_FLOOR
	INDICATOR_FLOOR
	BUTTON_STOP
	OBSTRUCTION_SENS
	DOOR_LAMP
)
