package main

import (
	"fmt"
	"hardware"
	"queue"
	"typedef"
	"time"
)

func main() {

	// Initialize the hardware module and the channel to message with it.
	buttonChannel := make(chan typedef.ButtonEvent, 1) // Channel to receive buttonEvents
	lightChannel := make(chan typedef.LightEvent, 1) // Channel to send LightEvents
	motorChannel := make(chan typedef.MotorEvent, 1) // Channel to send MotorEvents
	floorChannel := make(chan typedef.FloorEvent, 1) // Channel to receive floorEvents
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

	// Door timer
	doorTimer := time.NewTimer(time.Second)
	doorTimer.Stop()

	// ----------------------  WAIT FOR EVENTS! -------------------------
	for{
	select {
	case buttonEvent :=<- buttonChannel:
		if buttonEvent.ButtonType==typedef.BUTTON_CALL_UP || buttonEvent.ButtonType==typedef.BUTTON_CALL_DOWN || buttonEvent.ButtonType==typedef.BUTTON_COMMAND{
			fmt.Printf("NEW ORDER: floor: %d, buttonType: %d\n", buttonEvent.Floor, buttonEvent.ButtonType)
			queue.SetOrder(buttonEvent.Floor, buttonEvent.ButtonType, 1)
			queue.PrintQue()
			motorChannel <- typedef.MotorEvent{MotorDirection: typedef.DIR_UP}

		} else if buttonEvent.ButtonType == typedef.BUTTON_STOP {
			// Stop button has been pressed.
			motorChannel<-typedef.MotorEvent{MotorDirection: typedef.DIR_STOP}
			doorTimer.Reset(5*time.Second)
		} else if buttonEvent.ButtonType == typedef.OBSTRUCTION_SENS {
			// Obstruction
			motorChannel <- typedef.MotorEvent{MotorDirection: typedef.DIR_STOP}
		}
	case floorEvent :=<- floorChannel:
		fmt.Printf("At floor: %d\n", floorEvent.Floor)
		dir := hardware.CurrentDirection;
		if dir == -1 {
			dir = 0;
		} else if dir == 1{
			dir = 1;
		}
		if queue.CheckOrder(floorEvent.Floor, hardware.CurrentDirection) {
			motorChannel <- typedef.MotorEvent{MotorDirection: typedef.DIR_STOP}
			queue.SetOrder(floorEvent.Floor, dir, 0)
			queue.SetOrder(floorEvent.Floor, 3, 0)
			// TODO -> Do this better
			doorTimer.Reset(2*time.Second)
			fmt.Println("ONEELEVATOR:\t Closing the door for 2 seconds.")

		}
	case <- doorTimer.C:
		
		if hardware.CurrentFloor == -1 { // Stop button pressed.
			if hardware.PreviousDirection == 1{
				motorChannel<-typedef.MotorEvent{MotorDirection: typedef.DIR_UP}
			} else if hardware.PreviousDirection ==-1{
				motorChannel<-typedef.MotorEvent{MotorDirection: typedef.DIR_DOWN}
			} 
		}else if dir := queue.AnyOrders(hardware.PreviousFloor, hardware.PreviousDirection); dir != 0 { // The door is closed. Check if should proceed, or stay here.
			if dir == 1{
				motorChannel<-typedef.MotorEvent{MotorDirection: typedef.DIR_UP}
			} else if dir ==-1{
				motorChannel<-typedef.MotorEvent{MotorDirection: typedef.DIR_DOWN}
			} 
		}
	}
	}
}


