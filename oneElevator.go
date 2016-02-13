package main

import (
	"fmt"
	"typedef"
	"time"
	"hardware"
)


// Keep all relevant variables in a struct, split for networking with external orders in extended state?
type ElevatorState struct {
	Lastfloor int
	Direction int
	Moving bool
	OpenDoor bool
	InternalOrders[typedef.N_FLOORS]bool
	ExternalOrders [typedef.N_FLOORS][typedef.N_BUTTONS - 1]bool
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
	return state.haveOrderBelow() || state.haveOrderAbove() || state.haveOrdersAtCurrentFloor()
}

func (state *ElevatorState) haveOrderAbove() bool {
	for floor := typedef.N_FLOORS - 1; floor > state.Lastfloor; floor-- {
		if state.InternalOrders[floor] {
			return true
		}

		for _, order := range state.ExternalOrders[floor]{
			if order {
				return true
			}
		}
	}
	return false
}

func (state *ElevatorState) haveOrderBelow() bool {
	fmt.Println("")
	for floor := 0; floor < state.Lastfloor ; floor++ {
		if state.InternalOrders[floor] {
			return true
		}

		for _, order := range state.ExternalOrders[floor]{
			if order {
				return true
			}
		}
	}
	return false
}

func (state *ElevatorState) haveOrdersAtCurrentFloor() bool {
	if state.InternalOrders[state.Lastfloor]{
		return true
	}
	for _, order := range state.ExternalOrders {
		if order[0] && order[1] {
			return true
		}
	}
	return false
}

func (state *ElevatorState) printOrders() {
	for n, orders := range state.ExternalOrders {
		fmt.Printf("\t\t\t%t\t%t\t%t\n", orders[0], orders[1], state.InternalOrders[n])
	}
}

func (state *ElevatorState) printState() {
	fmt.Printf("\tElevatorState:\n\t\ttLastfloor: %d\tDirection: %d\t\n\t\tMoving: %t\tOpenDoor: %t\n", state.Lastfloor, state.Direction, state.Moving, state.OpenDoor)
	fmt.Printf("\tOrders: \n")
	state.printOrders()
}

func (state *ElevatorState) shouldStop() bool {
	if state.InternalOrders[state.Lastfloor] {
		return true
	}
	if state.Direction == typedef.DIR_DOWN {
		if state.ExternalOrders[state.Lastfloor][1] {
			return true
		}
	}
	if state.Direction == typedef.DIR_UP {
		if state.ExternalOrders[state.Lastfloor][0] {
			return true
		}
	}
	if !state.haveOrderAbove()&&state.ExternalOrders[state.Lastfloor][1]{
		return true
	}
	if !state.haveOrderBelow()&&state.ExternalOrders[state.Lastfloor][0]{
		return true
	}
	return false
}

func (state *ElevatorState) nextDirection() int{
	if state.Direction == typedef.DIR_UP && state.haveOrderAbove() {
		return typedef.DIR_UP
	} else if state.Direction == typedef.DIR_DOWN && state.haveOrderBelow() {
		return typedef.DIR_DOWN
	} else if state.haveOrderBelow() {
		return typedef.DIR_DOWN
	} else if state.haveOrderAbove() {
		return typedef.DIR_UP
	}
	return 0 // Returns 0 if there are no orders.
}


func main() {
	var myStateV ElevatorState
	myState := &myStateV
	for _, order := range myState.ExternalOrders {
		order[0] = false
		order[1] = false
	}
	for n,_ := range myState.InternalOrders {
		myState.InternalOrders[n] = false
	}
	myState.printState()
	// Initialize the hardware module and the channel to message with it.
	buttonChannel := make(chan hardware.ButtonEvent, 1) // Channel to receive buttonEvents
	lightChannel := make(chan hardware.LightEvent, 3) // Channel to send driver.LightEvents
	motorChannel := make(chan int, 1) // Channel to send MotorEvents
	floorChannel := make(chan hardware.FloorEvent, 1) // Channel to receive floorEvents
	doorTimer := time.NewTimer(5*time.Second) // Door timer
	doorTimer.Stop()
	polldelay := time.Duration(10*time.Millisecond)
	fmt.Println("ONEELEVATOR:\t Polling delay set to: %d", polldelay)

	defer func(){
		motorChannel <- typedef.DIR_STOP
	}()

	err := hardware.Init(buttonChannel, lightChannel, motorChannel, floorChannel, polldelay) // Starts the hardware polling loop.
	if err != nil {
		fmt.Println("Error initializing hardware..")
		return
	}

	// ----------------------  WAIT FOR EVENTS! -------------------------
	for{
	select {
	case buttonEvent :=<- buttonChannel:
		// A event has been sendt to us on the button channel.
		bType := buttonEvent.ButtonType
		if bType == typedef.BUTTON_CALL_UP || bType == typedef.BUTTON_CALL_DOWN || bType == typedef.BUTTON_COMMAND {
			fmt.Printf("ONEELEVATOR:\t New Order, floor: %d, buttonType: %d\n", buttonEvent.Floor, buttonEvent.ButtonType)
			runNow := !myState.haveOrders()
			fmt.Printf("ONEELEVATOR:\t Do we have other orders? \t %t\n", !runNow)
			lightChannel<- hardware.LightEvent{LightType: bType, Floor: buttonEvent.Floor, Value : true}
			if bType == typedef.BUTTON_COMMAND {
				myState.InternalOrders[buttonEvent.Floor] = true
			} else {
				myState.ExternalOrders [buttonEvent.Floor][bType] = true
			}
			if runNow {
				fmt.Println("RUNNOW: value - true")
				// We have no orders, exequte this one immediately.
				if buttonEvent.Floor > myState.Lastfloor {
					// We are going up!
					myState.Direction = typedef.DIR_UP
					myState.Moving = true
					motorChannel <-  typedef.DIR_UP
				} else if buttonEvent.Floor < myState.Lastfloor{
					// We are going down.
					myState.Direction = typedef.DIR_DOWN
					myState.Moving = true
					motorChannel <- typedef.DIR_DOWN
				} else {
					// Ordered at current floor.
					motorChannel <- typedef.DIR_STOP
					lightChannel <- hardware.LightEvent{LightType: typedef.DOOR_LAMP, Value: true}
					doorTimer.Reset(5*time.Second)
					myState.Moving = false
					myState.setOpenDoor(true)
				}
				
			}
			

		} else if bType == typedef.BUTTON_STOP {
			fmt.Printf("ONEELEVATOR:\t Received stop button event, value:\t%t\n", buttonEvent.Value)
			if buttonEvent.Value {
				motorChannel <- typedef.DIR_STOP
				lightChannel <- hardware.LightEvent{LightType: typedef.BUTTON_STOP, Value: true}
				myState.Moving = false
			} else {
				motorChannel <- myState.Direction
				lightChannel <- hardware.LightEvent{LightType: typedef.BUTTON_STOP, Value: false}
				myState.Moving = true
			}
			
		}
		myState.printState()
	case floorEvent:=<-floorChannel:
		fmt.Printf("ONEELEVATOR:\t At floor: %d, direction: %d\n", floorEvent.Floor, myState.Direction)
		myState.setLastFloor(floorEvent.Floor)
		// Initialization between floors.
		if myState.Direction == typedef.DIR_STOP {
			fmt.Printf("ONEELEVATOR:\t Stopping\n")
			motorChannel <- typedef.DIR_STOP
			myState.Moving = false
		}
		
		if myState.shouldStop(){
			fmt.Printf("ONEELEVATOR:\t Stopping at this floor, direction: %d opening door for 3 sec\n", myState.Direction)
			// Turn off lights and clear orders.
			if myState.Direction == typedef.DIR_UP{
				lightChannel <- hardware.LightEvent{LightType: typedef.BUTTON_CALL_UP, Floor: floorEvent.Floor, Value: false}
				lightChannel <- hardware.LightEvent{LightType: typedef.BUTTON_COMMAND, Floor: floorEvent.Floor, Value: false}
				myState.InternalOrders[floorEvent.Floor] = false
				myState.ExternalOrders[floorEvent.Floor][1] = false

				// Check if we are turning around.
				if b := myState.nextDirection(); b == typedef.DIR_DOWN || b == typedef.DIR_STOP {
					lightChannel <- hardware.LightEvent{LightType: typedef.BUTTON_CALL_DOWN, Floor: floorEvent.Floor, Value: false}
					myState.ExternalOrders[floorEvent.Floor][0] = false
				}


			} else if myState.Direction==typedef.DIR_DOWN{
				lightChannel <- hardware.LightEvent{LightType: typedef.BUTTON_CALL_DOWN, Value: false}
				lightChannel <- hardware.LightEvent{LightType: typedef.BUTTON_COMMAND, Value: false}
				myState.InternalOrders[floorEvent.Floor] = false
				myState.ExternalOrders[floorEvent.Floor][0] = false

				// Check if we are turning around.
				if b := myState.nextDirection(); b == typedef.DIR_UP || b == typedef.DIR_STOP {
					lightChannel <- hardware.LightEvent{LightType: typedef.BUTTON_CALL_UP, Floor: floorEvent.Floor, Value: false}
					myState.ExternalOrders[floorEvent.Floor][1] = false
				}
			}
			motorChannel <- typedef.DIR_STOP
			doorTimer.Reset(3*time.Second)
			lightChannel <- hardware.LightEvent{LightType: typedef.DOOR_LAMP, Value: true}
			myState.Moving = false
			myState.setOpenDoor(true)
		}
		myState.printState()

	case <- doorTimer.C:
		myState.setOpenDoor(false)
		lightChannel <- hardware.LightEvent{LightType: typedef.DOOR_LAMP, Value: false}
		fmt.Printf("ONEELEVATOR:\t Door timeout.\n")	
		if b := myState.haveOrders(); !b {
			fmt.Printf("ONEELEVATOR:\t No orders, staying at floor.\n")
			myState.setDirection(typedef.DIR_STOP)
		} else if myState.Direction == typedef.DIR_UP && myState.haveOrderAbove() {
			fmt.Println("ONEELEVATOR:\t Have order above, and direction upwards.. Continuing.")
			myState.setMoving(true)
			motorChannel <- typedef.DIR_UP
		} else if myState.Direction == typedef.DIR_DOWN && myState.haveOrderBelow() {
			fmt.Println("ONEELEVATOR:\t Have order above, and direction downwards.. Continuing.")
			myState.setMoving(true)
			motorChannel <- typedef.DIR_DOWN
		} else if myState.haveOrderBelow() {
			fmt.Println("ONEELEVATOR:\t Have order below.. Go down.")
			myState.setMoving(true)
			motorChannel <- typedef.DIR_DOWN
			myState.setDirection(typedef.DIR_DOWN)

		} else if  myState.haveOrderAbove() {
			fmt.Println("ONEELEVATOR:\t Have order above.. Go up.")
			myState.setMoving(true)
			motorChannel <- typedef.DIR_UP
			myState.setDirection(typedef.DIR_UP)
		}
	}
	myState.printState()
}
}


