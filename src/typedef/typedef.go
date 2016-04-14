package typedef

import(
 "time"
 "fmt"
 )

// Elevator system constants
const N_FLOORS int = 4 
const N_BUTTONS int = 3


// --------------------- "Enumerators" --------------------
// Motor directions
const(
	DIR_DOWN  = -1
	DIR_STOP = 0
	DIR_UP = 1
)

// Hardware events
const (
	BUTTON_CALL_DOWN = iota
	BUTTON_CALL_UP
	BUTTON_COMMAND
	BUTTON_STOP
	SENSOR_FLOOR
	INDICATOR_FLOOR
	OBSTRUCTION_SENS
	DOOR_LAMP
)

// System events
const(
	// Events that may be received in ExtOrderStruct
	EventSendOrderToElevator = iota
	EventAccOrderFromElevator
	EventConfirmAccFromElevator
	
	// Events that may be received in ExtBackupStruct
	EventSendBackupToAll
	EventRequestStateFromElevator
	EventStillOnline
	EventAccBackup
	EventBackupAtAllConfirmed
	EventAnsweringBackupRequest
	
	// Hardware events
	EventButtonPressed
	EventFloorReached
)

// // --------------------- "String arrays" --------------------
var HardwareEventType = []string{
	"BUTTON_CALL_DOWN",
	"BUTTON_CALL_UP",
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
	"EventStillOnline",
	"EventAccBackup",
	"EventBackupAtAllConfirmed",
	"EventAnsweringBackupRequest",
	"EventButtonPressed",
	"EventFloorReached",
}

var MotorDirections = []string {
	"DOWN",
	"STOP",
	"UP",
}


// --------------------- "Data structures" --------------------

type HardwareEvent struct{
	LightType int
	Floor int
	Value bool
	ButtonType int
	CurrentDirection int
	Event int
}

// Elevators state
type StateStruct struct {
	LocalIP string
	InternalOrders [N_FLOORS]bool
	ExternalOrders [2][N_FLOORS] bool
	PrevFloor int // This is the latest valid floor
	CurrentDirection int
	Moving bool
	OpenDoor bool
}

type OrderStruct struct{
	OrderID int
	Floor int
	Type int
	ReceivedTime time.Time
	DispatchedTime time.Time
	Status int
}

type ExtOrderStruct struct {
	Order OrderStruct
	OrderID int
	SendTo string 
	SentFrom string 
	Event int
} 

type BackupStruct struct {
	CurrentState StateStruct
	BackupTime time.Time // Time of creation, i.e. the time it is received
}

type ExtBackupStruct struct {
	BackupData BackupStruct
	SentFrom string
	SendTo string
	Event int 
}

type Elevator struct{
	State StateStruct
	Time time.Time
}


/*
	Function to see if backup is valid.
	If event it EventSendBackupToAll, check the size of the queues.
	Always check that sender IP is different from localIP, so we 
	don`t listen to our own messages.
*/
func (e ExtBackupStruct) Valid(localIP string) bool {
	if e.Event == EventSendBackupToAll{
		ex := e.BackupData.CurrentState.ExternalOrders
		in := e.BackupData.CurrentState.InternalOrders
		if len(ex[0]) != N_FLOORS || len(ex[1]) != N_FLOORS || len(in) != N_FLOORS {
			return false
		}
	}
	if e.SentFrom  == localIP {
		return false
	}
	if e.SentFrom == ""{
		return false
	}
	return true
}

func (o ExtOrderStruct) Valid() bool {
	// TODO -> Actually implement an acceptance test to see if the order is valid.
	return true
}


func (e *Elevator) ShouldStop() bool {
	floor := e.State.PrevFloor

	switch e.State.CurrentDirection{
	case DIR_STOP:
		return true;
	case DIR_UP:
		return !e.State.OrdersAbove() || e.State.ExternalOrders[BUTTON_CALL_UP][floor] || e.State.InternalOrders[floor] || floor == N_FLOORS-1
	case DIR_DOWN:
		return !e.State.OrdersBelow() || e.State.ExternalOrders[BUTTON_CALL_DOWN][floor] || e.State.InternalOrders[floor] || floor == 0
	}
	return true
}

func (e *Elevator) SetDirection(dir int){
	e.State.CurrentDirection = dir
}

func(e *Elevator) setDoor(b bool){
	e.State.OpenDoor = b
}

func (e *Elevator) DoorOpen() bool {
	return e.State.OpenDoor
}

func (e *Elevator) IsMoving() bool{
	return e.State.Moving
}

func (e *Elevator) SetMoving(b bool){
	e.State.Moving = b
}

func (e *Elevator) GetFloor() int {
	return e.State.PrevFloor
}

func (e *Elevator) SetFloor(f int) {
	e.State.PrevFloor = f
}

func (e *Elevator) GetDirection() int {
	return e.State.CurrentDirection
}

func (e *Elevator) GetNextDirection() int {
	if !e.State.HaveOrders() {
		return DIR_STOP
	}

	switch e.State.CurrentDirection {
	case DIR_DOWN:
		if e.State.OrdersBelow() && e.State.PrevFloor != 0{
			return DIR_DOWN
		}
	case DIR_UP:
		if e.State.OrdersAbove() && e.State.PrevFloor != N_FLOORS - 1{
			return DIR_UP
		}
	case DIR_STOP:
		if e.State.OrdersAbove() {
			return DIR_UP
		} else if e.State.OrdersBelow() {
			return DIR_DOWN
		}
	}
	return DIR_STOP

}


func (e *Elevator) SetInternalOrder(floor int, value bool){
	e.State.InternalOrders[floor] = value
}
/*  This function is purely to ease the visualization of the system,
	and is poorly maintanable, due to the hardcoded number of floors.
*/

func (e *Elevator) MakeQueue() string{
	
	in := e.State.InternalOrders
	ex := e.State.ExternalOrders

	inTemp := [N_FLOORS]string{}
	exTempUp := [N_FLOORS]string{}
	exTempDown := [N_FLOORS]string{}
	for indx, val := range in{
		if val{
			inTemp[indx] = "x"
		} else {
			inTemp[indx] = "-"
		}
	}

	for indx, val := range ex[0]{
		if val{
			exTempDown[indx] = "x"
		} else {
			exTempDown[indx] = "-"
		}
	}

	for indx, val := range ex[1]{
		if val{
			exTempUp[indx] = "x"
		} else {
			exTempUp[indx] = "-"
		}
	}

	str:= "----------------------------------\n"
	str+= "\t\t    | Floor: |  0  |  1  |  2  |  3  |\n"
	str+= "\t\t    ----------------------------------\n"
	str+= fmt.Sprintf("\t\t    | UP:    |  %s  |  %s  |  %s  |     |\n", exTempUp[0], exTempUp[1], exTempUp[2])
	str+= "\t\t    ----------------------------------\n"
	str+= fmt.Sprintf("\t\t    | DOWN:  |     |  %s  |  %s  |  %s  |\n", exTempDown[1], exTempDown[2], exTempDown[3])
	str+= "\t\t    ----------------------------------\n"
	str+= fmt.Sprintf("\t\t    | Cab:   |  %s  |  %s  |  %s  |  %s  |\n", inTemp[0], inTemp[1], inTemp[2], inTemp[3])
	
	return str
}


func (s StateStruct) OrdersAbove() bool {
	for floor := N_FLOORS -1;  floor > s.PrevFloor; floor-- {
		if s.InternalOrders[floor]{
			return true
		}
		if s.ExternalOrders[0][floor] || s.ExternalOrders[1][floor]{
			return true
		}
	}
	return false
}

func (s StateStruct) OrdersBelow() bool {
	for floor := 0; floor < s.PrevFloor; floor ++ {
		if s.InternalOrders[floor] {
			return true
		}
		if s.ExternalOrders[0][floor] || s.ExternalOrders[1][floor]{
			return true
		}
	}
	return false
}


func (s StateStruct) OrderAtCurrentFloor() bool {
	floor := s.PrevFloor
	return s.InternalOrders[floor] || s.ExternalOrders[0][floor] || s.ExternalOrders[1][floor]
}

func (s StateStruct) HaveOrders() bool {
	return s.OrdersBelow() || s.OrdersAbove() || s.OrderAtCurrentFloor()
}

func MakeBackupMessage(e *Elevator) ExtBackupStruct {
	return ExtBackupStruct{BackupData: BackupStruct{CurrentState: e.State}, Event: EventSendBackupToAll}
}



















