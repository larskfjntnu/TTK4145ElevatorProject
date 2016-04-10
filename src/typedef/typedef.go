package typedef

include(
	"time"
)

const N_FLOORS int = 4 // TODO -> Do this dynamically.
const N_BUTTONS int = 3


// --------------------- "Enumerators" --------------------

const(
	DIR_DOWN  = -1
	DIR_STOP = 0
	DIR_UP = 1
)

// ------------- Enumerators
// Hardware events
const (
	BUTTON_CALL_UP = iota
	BUTTON_CALL_DOWN
	BUTTON_COMMAND
	BUTTON_STOP
	SENSOR_FLOOR
	INDICATOR_FLOOR
	OBSTRUCTION_SENS
	DOOR_LAMP
)

// Events OrderEvents are 0-4
const(
	// Events that may be received in ExtOrderStruct
	EventSendOrderToElevator = iota
	EventAccOrderFromElevator
	EventConfirmAccFromElevator
	
	// Events that may be received in ExtBackupStruct
	EventSendBackupToAll
	EventRequestStateFromElevator
	
	// Internal order event
	EventReceivedOrderFromElevator
	
	// Internal backup event
	EventReceivedBackupFromElevator
	EventNoBackupWithinLimit
	
	// HW events
	EventButtonPressed
	EventFloorReached
	EventSetMotor
	EventSetLight
	
)

// Order status
const (
	Waiting = iota
	Executing
)

// ------------- String arrays for debugging.
var HardwareEventType = []string{
	"BUTTON_CALL_UP",
	"BUTTON_CALL_DOWN",
	"BUTTON_COMMAND",
	"BUTTON_STOP",
	"SENSOR_FLOOR",
	"INDICATOR_FLOOR",
	"OBSTRUCTION_SENS",
	"DOOR_LAMP",
}

var EventType = []string{
	"EventSendOrderToElevator",
	"EventAccOrderFromElevator",
	"EventConfirmAccFromElevator",
	"EventSendBackupToAll",
	"EventRequestStateFromElevator",
	"EventReceivedOrderFromElevator"
	"EventReceivedBackupFromElevator",
	"EventNoBackupWithinLimit",
	"EventButtonPress",
	"EventFloorReached",
	"EventSetMotor",
	"EventSetLight",
}

var OrderStatus = []string{
	"Waiting",
	"Executing"
}

var MotorDirections = []string {
	"DOWN"
	"STOP"
	"UP"
}


// ------------- Data structures

// Hardware structs

type HardwareEvent struct{
	LightType int
	Floor int
	Value bool
	MotorDirection int
	ButtonType int
	CurrentDirection int
	Event int
}

// The elevators state
type StateStruct struct {
	LocalIP string
	InternalOrders [N_FLOORS]bool
	ExternalOrders [N_FLOORS][2] bool
	PrevFloor int
	CurrentDirection int
	Moving bool
	OpenDoor bool
}

type OrderStruct struct{
	OrderID int
	Floor int
	Type int
	ReceviedTime time.Time
	Status int
}

// Extended Order struct to be sent to the appropriate elevator
type ExtendedOrderStruct struct {
	Order OrderStruct
	SendTo string // IP
	SentFrom string // IP
	Event int
} 

type BackupStruct struct {
	CurrentState StateStruct
	BackupTime time.Time // Time of creation, i.e. the time the backup is saved in other prosess
}

// Extended Backup struct to to others
type ExtBackupStruct struct {
	State BackupStruct
	RequesterIP string // only for requesting state
	Event int // Categorizes the type of message. 'alive', 'request' or 'answertorequest'
}

type Elevator struct{
	State StateStruct
	Time time.Time
}


func MakeBackupMessage(e *Elevator) BackupStruct {
	return BackupStruct{State: e.State, Event: EventBackup}
}

func (e Elevator) ShouldStop() bool {
	floor := e.State.PrevFloor

	switch e.State.CurrentDirection{
	case DIR_STOP:
		return true;
	case DIR_UP:
		return !e.OrdersAbove() || e.ExternalOrders[floor][BUTTON_CALL_UP] ||
			    e.InternalOrders[floor] || floor == N_FLOORS-1
	case DIR_DOWN:
		return !e.OrdersBelow() || e.ExternalOrders[floor][BUTTON_DIR_DOWN ||
				e.InternalOrders[floor] || floor == 0
	}
	return true
}

func (e Elevator) GetNextDirection() int {
	if !e.HaveOrders() {
		return DIR_STOP
	}

	switch e.State.Direction {
	case DIR_DOWN:
		if e.OrdersBelow() && e.State.PrevFloor != 0{
			return DIR_DOWN
		}
		falltrough
	case DIR_UP:
		if e.OrdersAbove() && e.State.PrevFloor != N_FLOORS - 1{
			return DIR_UP
		}
		falltrough
	case STOP:
		if e.OrdersAbove() {
			return DIR_UP
		} else if e.OrdersBelow() {
			return DIR_DOWN
		}
	}
	return STOP

}

func (s StateStruct) OrdersAbove() bool {
	for floor := N_FLOORS -1;  floor > s.PrevFloor; floor-- {
		if s.InternalOrders[floor]{
			return true
		}
		for _, order := range e.State.ExternalOrders[floor] {
			if order {
				return true
			}
		}
	}
	return false
}

func (s StateStruct) OrdersBelow() bool {
	for floor := 0; floor < s.PrevFloor; floor ++ {
		if s.InternalOrders[floor] {
			return true
		}
		for _, order := range e.State.ExternalOrders[floor] {
			if order {
				return true
			}
		}
	}
	return false
}

func (s StateStruct) OrderAtFloor() bool {
	floor := s.PrevFloor
	return s.InternalOrders[floor] || s.ExternalOrders[floor][0] || s.ExternalOrders[floor][1]
}

func (s StateStruct) HaveOrders() bool {
	return OrdersBelow() || OrdersAbove() || OrderAtFloor
}



















