package typedef

const N_FLOORS int = 4 // TODO -> Do this dynamically.
const N_BUTTONS int = 3


// --------------------- "Enumerators" --------------------

const(
	DIR_DOWN int = -1 << iota 
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

// Events
const(
	EventNotifyAlive = iota
	EventBackup
	EventRequestState
	EventReturnRestoredState
	EventNewOrder
	EventConfirmOrder
	EventAcknowledgeConfirmedOrder
	EventOrderDone
	EventAcknowledgeOrderDone
	EventReassignOrder
)

// Order status
const (
	InActive = iota
	Waiting
	Executing
)


