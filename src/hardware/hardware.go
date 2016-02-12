package hardware
/*
	This is the module wich interact with the hardware of the elevator.
	It runs as a separate goroutine wich continously 'pings' the hardware
	for new information. It contains a continous loop which querys the 
	hardware for status of the order buttons and the floor sensors.
	If it detects some new information, it calls the main module to handle it.
	It also contains an interface where the main module can call the hardware
	module to set the lights to indicate wether a floor is ordered or if the
	order is completed and to set the engine direction(which means it goes up,
	down or stops.).
	It contains two internal functions which are used to simplify the main
	loop. These functions check the button status' and floor status' 
	respectively. 
*/

//		--------------------------------------------------------------------

// TODO -> Define fault modes and fault tolerance.
/*
	We need to implement all the acceptance tests as well as other
	ways to support the fault tolerance.
*/

//		---------------------------------------------------------------------

/*
	Need to implement the driver source. These files are C code. 
	Go recognices the 'import' statement within the comment and lets
	us reference the functions in the interface of the C code in the
	Go source code. The 'import "C" ' statement is a 'pseudo package' which
	let cgo recognise the C namespace. 
	The 'import "unsafe" ' is needed because the memory allocations made by
	C are not known to the Go memory manager. When C creates a string or such,
	we need to free this by calling C.free
*/
/*
	#import driver/io.h
*/
import (
	"C"
	"typedef"
	"driver"
	"fmt"
	)

var lightChannelMatrix [][]int
var buttonChannelMatrix [][]int

var PreviousFloor int
var CurrentFloor int
var CurrentDirection int
var PreviousDirection int
var initialized bool = false
const motorspeed = 2800
//		--------------------------------------------------------------------


func Init(buttonChannel chan<- typedef.ButtonEvent, lightChannel <-chan typedef.LightEvent, motorChannel <-chan typedef.MotorEvent, floorChannel chan<- typedef.FloorEvent) error{
	if initialized{
		return fmt.Errorf("Hardware is already initialized.")
	}
	initSuccess := driver.IOInit()
	if initSuccess!=nil{
		return fmt.Errorf("Unable to initialize hardware.")
	}
	row1 := []int{driver.LIGHT_UP1, driver.LIGHT_DOWN1, driver.LIGHT_COMMAND1}
	row2 := []int{driver.LIGHT_UP2, driver.LIGHT_DOWN2, driver.LIGHT_COMMAND2}
	row3 := []int{driver.LIGHT_UP3, driver.LIGHT_DOWN3, driver.LIGHT_COMMAND3}
	row4 := []int{0, driver.LIGHT_DOWN4, driver.LIGHT_COMMAND4}
	lightChannelMatrix = append(lightChannelMatrix, row1)
	lightChannelMatrix = append(lightChannelMatrix, row2)
	lightChannelMatrix = append(lightChannelMatrix, row3)
	lightChannelMatrix = append(lightChannelMatrix, row4)

	row1 = []int{driver.BUTTON_UP1, 0, driver.BUTTON_COMMAND1}
	row2 = []int{driver.BUTTON_UP2, driver.BUTTON_DOWN2, driver.BUTTON_COMMAND2}
	row3 = []int{driver.BUTTON_UP3, driver.BUTTON_DOWN3, driver.BUTTON_COMMAND3}
	row4 = []int{0, driver.BUTTON_DOWN4, driver.BUTTON_COMMAND4}
	buttonChannelMatrix = append(buttonChannelMatrix, row1)
	buttonChannelMatrix = append(buttonChannelMatrix, row2)
	buttonChannelMatrix = append(buttonChannelMatrix, row3)
	buttonChannelMatrix = append(buttonChannelMatrix, row4)
	for f:=0;f<typedef.N_FLOORS;f++{
		for b:=typedef.BUTTON_CALL_UP;b<typedef.N_BUTTONS;b++{
			setButtonLight(f, b, false)
		}
	}
	// Start goroutines to handle hardware events.
	go readButtons(buttonChannel)
	go readFloorSensors(floorChannel)
	go setLights(lightChannel)
	go motorControl(motorChannel)
	return nil
	// TODO -> Acceptance test!!!
}


// This function runs continously as a goroutine, pinging the hardware for button presses.
func readButtons(buttonChannel chan<- typedef.ButtonEvent){
	
	// This while loop runs continously, pinging the hardware for button presses.
	for {
		// Check if there are any new orders(buttons pressed).
		for floor := 0; floor < typedef.N_FLOORS; floor ++ {
			for buttonType := typedef.BUTTON_CALL_UP; buttonType < typedef.N_BUTTONS; buttonType++ {
				if checkButtonPressed(buttonType, floor) != 0{
					// Pass a hardwareevent to the event channel.
					buttonChannel <- typedef.ButtonEvent{ButtonType: buttonType, Floor: floor}
				}
			}
		}
		if checkStopSignal() {
			buttonChannel <- typedef.ButtonEvent{ButtonType: typedef.BUTTON_STOP}
		}
		if checkObstructionSignal() {
			buttonChannel <- typedef.ButtonEvent{ButtonType: typedef.OBSTRUCTION_SENS}
		}
	}
}

// This function runs continously as a goroutine, pinging the hardware for floor arrivals.
func readFloorSensors(floorChannel chan<- typedef.FloorEvent) {
	for {
		floor := checkFloor()
		if floor != CurrentFloor {
			PreviousFloor = CurrentFloor
			CurrentFloor = floor
			if  floor > -1 {
				floorChannel <- typedef.FloorEvent{CurrentDirection: typedef.Direction(CurrentDirection), Floor: floor} 
			}	
		}
	}
}
// This function runs continously as a goroutine, waiting for orders to set lights.
func setLights(lightChannel <-chan typedef.LightEvent){
	select{
		case lightEvent:=<-lightChannel:
			switch lightEvent.LightType{
			case typedef.BUTTON_CALL_UP:
				setButtonLight(lightEvent.Floor, 0, lightEvent.Value)
			case typedef.BUTTON_CALL_DOWN:
				setButtonLight(lightEvent.Floor, 1, lightEvent.Value)
			case typedef.BUTTON_COMMAND:
				setButtonLight(lightEvent.Floor, 2, lightEvent.Value)
			case typedef.INDICATOR_FLOOR:
				setFloorIndicator(lightEvent.Floor)
			case typedef.BUTTON_STOP:
				setStopLamp(lightEvent.Value)
			case typedef.DOOR_LAMP:
				setDoorLamp(lightEvent.Value)
			default:
				// Do some error handling.
			}
	}
}

// This function runs continously as a goroutine, waiting for orders to set the motor direction
func motorControl(motorChannel<-chan typedef.MotorEvent){
	motorEv :=<-motorChannel
	setMotorDirection(motorEv.MotorDirection)
}
/*
	This functions loops through the different types of buttons at all the
	floors and checks if any buttons are pressed.
*/
func checkButtonPressed(buttonType, floor int) (pressed int){
	// TODO -> Do this better in terms of counter variable names and button types.
	if driver.IOReadBit(buttonChannelMatrix[floor][buttonType]){
		return 1
	} else {
		return 0
	}
}

/*
	This functions checks the sensor at a
	given floor to see if the elevator is at that floor.
*/
func checkFloor() (floor int){
	if driver.IOReadBit(driver.SENSOR_FLOOR1) {
		return 0
	} else if driver.IOReadBit(driver.SENSOR_FLOOR2) {
		return 1
	} else if driver.IOReadBit(driver.SENSOR_FLOOR3) {
		return 2
	} else if driver.IOReadBit(driver.SENSOR_FLOOR4) {
		return 3
	} else {
		return -1
	}
}

/*
	This function checks the status of the stop button
*/
func checkStopSignal() bool {
	return driver.IOReadBit(driver.STOP)
}

/*
	This function checks the status of the obstruction button/signal
*/
func checkObstructionSignal() bool {
	return driver.IOReadBit(driver.OBSTRUCTION)
}

/*
	This function/channel (called from another goroutine) sets the light of a 
	specific type at the given floor to the specified value.
*/
func setButtonLight(floor, buttonType int, value bool) error {
	// TODO -> Some acceptance test for the arguments..
	if value {
		driver.IOSetBit(lightChannelMatrix[floor][buttonType])
	} else {
		driver.IOSetBit(lightChannelMatrix[floor][buttonType])
	}
	return nil
}

/*
	This function/channel(called from another goroutine) sets the direction of
	the motor(any other direction than 0/STOP means it will run in this direction
	immediately).
*/
func setMotorDirection(direction typedef.Direction) error {
	if direction == 0 {
		driver.IOWriteAnalog(driver.MOTOR, 0)
	} else if direction > 0 {
		driver.IOClearBit(driver.MOTORDIR)
		driver.IOWriteAnalog(driver.MOTOR, motorspeed)
	} else if direction < 0 {
		driver.IOClearBit(driver.MOTORDIR)
		driver.IOWriteAnalog(driver.MOTOR, motorspeed)
	}
	PreviousDirection = CurrentDirection
	if direction == typedef.DIR_DOWN {
		CurrentDirection = -1
	} else if direction == typedef.DIR_STOP {
		CurrentDirection = 0
	} else {
		CurrentDirection = 1
	}

	// TODO -> Do some acceptance test to see if the direction was set.
	// return some error.New()..
	// If acceptance test completes.
	return nil
}

/*
	This function sets the indicator at a given floor.
*/
func setFloorIndicator(floor int) {
	// Binary encoding, one light is always on 00, 01, 10 or 11
	switch floor {
	case 1:
		driver.IOClearBit(driver.LIGHT_FLOOR_IND1)
		driver.IOClearBit(driver.LIGHT_FLOOR_IND2)
	case 2:
		driver.IOSetBit(driver.LIGHT_FLOOR_IND1)
		driver.IOClearBit(driver.LIGHT_FLOOR_IND2)
	case 3:
		driver.IOClearBit(driver.LIGHT_FLOOR_IND1)
		driver.IOSetBit(driver.LIGHT_FLOOR_IND2)
	case 4:
		driver.IOSetBit(driver.LIGHT_FLOOR_IND1)
		driver.IOSetBit(driver.LIGHT_FLOOR_IND2)
	}
}

/*
	This function sets the value of the door lamp
*/	
func setDoorLamp(value bool) {
	if value {
		driver.IOSetBit(driver.LIGHT_DOOR_OPEN)
	} else {
		driver.IOClearBit(driver.LIGHT_DOOR_OPEN)
	}
}

/*
	This function sets the value of the stop lamp.
*/
func setStopLamp(value bool) {
	if value {
		driver.IOSetBit(driver.LIGHT_STOP)
	} else {
		driver.IOClearBit(driver.LIGHT_STOP)
	}
}















