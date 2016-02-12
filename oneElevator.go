package main

import (
	"fmt"
	"hardware"
	"queue"
	"typedef"
)

func main() {

	// Initialize the hardware module and the channel to message with it.
	buttonChannel := make(<-chan typedef.ButtonEvent) // Channel to receive buttonEvents
	lightChannel := make(chan<- typedef.LightEvent) // Channel to send LightEvents
	motorChannel := make(chan<- typedef.MotorEvent) // Channel to send MotorEvents
	floorChannel := make(<-chan typedef.FloorEvent) // Channel to receive floorEvents
	err := hardware.Init(buttonChannel, lightChannel) // Starts the hardware polling loop.
	if err != nil {
		fmt.Println("Error initializing hardware..")
		return
	}
	err = queue.Init()
	if err != nil {
		fmt.Println("Error initializing queue..")
		return
	}

	// Door timer
	doorTimer := time.NewTimer(seconds)
	doorTimer.Stop()

	// ----------------------  WAIT FOR EVENTS! -------------------------
	select {
	case buttonEvent <- buttonChannel:
		if buttonEvent.ButtonType == BUTTON_CALL_UP || BUTTON_CALL_DOWN || BUTTON_COMMAND {
			queue.setOrder(buttonEvent.Floor, buttonEvent.ButtonType, buttonEvent.Value)
		} else if buttonEvent.ButtonType == BUTTON_STOP {
			// Stop button has been pressed.
			motorChannel <- MotorEvent(Direction: DIR_STOP)
			doorTimer.Reset(5*time.Second)
		} else if buttonEvent.ButtonType == OBSTRUCTION_SENS {
			// Obstruction
			motorChannel <- MotorEvent(Direction: DIR_STOP)
		}
	case floorEvent <- floorChannel:
		dir := hardware.CurrentDirection;
		if dir == -1 {
			dir = 0;
		} else if dir == 1{
			dir = 1;
		}
		if checkOrder(floorEvent.Floor, hardware.CurrentDirection) {
			motorChannel <- MotorEvent(Direction: DIR_STOP)
			setOrder(floorEvent.Floor, dir, 0)
			setOrder(floorEvent.Floor, 3, 0)
			// TODO -> Do this better
			doorTimer.Reset(2*time.Second)
			fmt.println("ONEELEVATOR:\t Closing the door for 2 seconds.")

		}
	case <- doorTimer.C
		
		if CurrentFloor == -1 { // Stop button pressed.
			motorChannel <- PreviousDirection
		}else if dir := anyOrders(PreviousFloor); dir != 0 { // The door is closed. Check if should proceed, or stay here.
			motorChannel <- dir
		}
	}

}
