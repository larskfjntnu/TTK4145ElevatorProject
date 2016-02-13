package main

import (
	"fmt"
	"hardware"
	"queue"
	"typedef"
	"time"
)


// Keep all relevant variables in a struct, split for networking with external orders in extended state?
type ElevatorState struct {
	Lastfloor int
	Direction int
	Moving bool
	OpenDoor bool
	InternalOrders[N_FLOORS]bool
	ExternalOrders [N_FLOORS][N_BUTTONS - 1]bool
}

func (state *ElevatorState) setDirection(dir int) {
	state.Direction = dir
}

func (state *ElevatorState) setMoving(moving bool) {
	state.Moving = moving
}

func (state *ElevatorState) setOpenDoor(open bool) {
	state.OpenDoor = open
}

func (state *ElevatorState) setLastFloor(floor int) {
	state.Lastfloor = floor
}

func (state *ElevatorState) haveOrders() bool {
	return state.haveOrderBelow() || state.haveOrdersAbove || state.haveOrdersAtCurrentFloor()
}

func (state *ElevatorState) haveOrderAbove() bool {
	for floor := N_FLOORS - 1; floor > state.CurrentFloor; floor-- {
		if state.InternalOrders[floor] == 1{
			return true
		}

		for _, order := range state.ExternalOrders[floor]{
			if order == 1 {
				return true
			}
		}
	}
	return false
}

func (state *ElevatorState) haveOrderBelow() bool {
	for floor := 0; floor < state.CurrentFloor; floor++ {
		if state.InternalOrders[floor] == 1{
			return true
		}

		for _, order := range state.ExternalOrders[floor]{
			if order == 1 {
				return true
			}
		}
	}
	return false
}

func (state *ElevatorState) haveOrdersAtCurrentFloor() bool {
	if state.InternalOrders[state.CurrentFloor] == 1 {
		return true
	}
	for _, order := range state.ExternalOrders {
		if order == 1 {
			return true
		}
	}
	return false
}



func main() {
	var myState ElevatorState
	// Initialize the hardware module and the channel to message with it.
	buttonChannel := make(chan ButtonEvent, 1) // Channel to receive buttonEvents
	lightChannel := make(chan LightEvent, 1) // Channel to send LightEvents
	motorChannel := make(chan MotorEvent, 1) // Channel to send MotorEvents
	floorChannel := make(chan FloorEvent, 1) // Channel to receive floorEvents
	doorTimer := time.NewTimer(time.Second) // Door timer
	doorTimer.Stop()

	err := hardware.Init(buttonChannel, lightChannel, motorChannel, floorChannel) // Starts the hardware polling loop.
	if err != nil {
		fmt.Println("Error initializing hardware..")
		return
	}
	err = queue.Init(4,3)
	if err != nil {
		fmt.Println("Error initializing queue..")
		return
	}

	// ----------------------  WAIT FOR EVENTS! -------------------------
	for{
	select {
	case buttonEvent :=<- buttonChannel:
		// A event has been sendt to us on the button channel.
		bType = buttonEvent.ButtonType
		if bType == BUTTON_CALL_UP || bType == BUTTON_CALL_DOWN || bType == BUTTON_COMMAND {
			fmt.Printf("ONEELEVATOR:\t New Order, floor: %d, buttonType: %d\n", buttonEvent.Floor, buttonEvent.ButtonType)
			runNow := !myState.haveOrders()
			if bType == BUTTON_COMMAND {
				myState.InternalOrders[buttonEvent.Floor] = 1
			} else {
				myState.ExternalOrders [buttonEvent.Floor][bType] = 1
			}
			if runNow {
				// We have no orders, exequte this one immediately.
				if buttonEvent.Floor > myState.Lastfloor {
					// We are going up!
					myState.Direction = DIR_UP
					myState.Moving = true
					motorChannel <-  MotorEvent{MotorDirection: DIR_UP}
				} else if {
					// We are going down.
					myState.Direction = DIR_DOWN
					myState.Moving = true
					motorChannel <- MotorEvent{MotorDirection: DIR_UP}
				} else {
					// Ordered at current floor.
					motorChannel <- MotorEvent{MotorDirection: DIR_STOP}
					lightChannel <- LighEvent{LightType: DOOR_LAMP, Value: true}
					doorTimer.Reset(3)
					myState.Moving = false
					myState.setOpenDoor(true)
				}
				
			}
			

		} else if bType == BUTTON_STOP {
			fmt.Printf("ONEELEVATOR:\t Received stop button event, value:\t%t", buttonEvent.Value)
			if buttonEvent.Value {
				motorChannel <- MotorEvent{MotorDirection: DIR_STOP}
				lightChannel <- LightEvents{LightType: BUTTON_STOP, Value: true}
				myState.Moving = false
			} else {
				motorChannel <- MotorEvent{MotorDirection: myState.Direction}
				lightChannel <- LightEvents{LightType: BUTTON_STOP, Value: false}
				myState.Moving = true
			}
			
		}

	case floorEvent :=<- floorChannel:
		fmt.Printf("ONEELEVATOR:\t At floor: %d, direction: %d\n", floorEvent.Floor, myState.Direction)
		myState.setLastFloor(floorEvent.Floor)
		if myState.Direction == DIR_UP && myState.haveOrderAbove() || myState.Direction == DIR_DOWN && myState.haveOrderBelow() || myState.InternalOrders[myState.Lastfloor] == 1{
			fmt.Printf("ONEELEVATOR:\t Stopping at this floor, opening door \n")
			motorChannel <- MotorEvent{MotorDirection: DIR_STOP}
			doorTimer.Reset(3)
			myState.Moving = false
			myState.setOpenDoor(true)
		}

	case <- doorTimer.C:
		myState.setOpenDoor(false)
		lightChannel <- LighEvent{LightType: DOOR_LAMP, Value: false}
		fmt.Printf("ONEELEVATOR:\t Door timeout.\n")	
		if !myState.haveOrders {
			fmt.Printf("ONEELEVATOR:\t No orders, staying at floor.\n")
		} else if myState.Direction == DIR_UP && myState.haveOrderAbove {
			fmt.Printf("ONEELEVATOR:\t Have order above, and direction upwards.. Continuing.")
			myState.setMoving = true
			motorChannel <- motorChannel <- MotorEvent{MotorDirection: DIR_UP}
		} else if myState.Direction == DIR_DOWN && myState.haveOrderBelow {
			fmt.Printf("ONEELEVATOR:\t Have order above, and direction downwards.. Continuing.")
			myState.setMoving = true
			motorChannel <- motorChannel <- MotorEvent{MotorDirection: DIR_DOWN}
		}
	}
}


