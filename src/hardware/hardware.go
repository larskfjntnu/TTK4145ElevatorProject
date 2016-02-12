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
	"unsafe"
	"channels"
	"typedef"
	"driver"
	)

var lightChannelMatrix [][]int
var buttonChannelMatrix [][] int

var PreviousFloor int
var CurrentFloor int
var CurrentDirection int
var PreviousDirection int
var initialized boolean = false
const motorspeed = 2800
//		--------------------------------------------------------------------


func Init(buttonChannel chan<- typedef.ButtonEvent, lightChannel <-chan typedef.LightEvent, motorChannel <-chan typedef.MotorEvent, floorChannel chan<- typedef.FloorEvent) error {
	if initialized{
		return new.Error("Hardware is already initialized.")
	}
	initSuccess := driver.IOInit()
	if(init_success != 1) {
		return new.Error("Unable to initialize hardware.")
	}
	row1 := {channels.LIGHT_UP1, channels.LIGHT_DOWN1, channels.LIGHT_COMMAND1}
	row2 := {channels.LIGHT_UP2, channels.LIGHT_DOWN2, channels.LIGHT_COMMAND2}
	row3 := {channels.LIGHT_UP3, channels.LIGHT_DOWN3, channels.LIGHT_COMMAND3}
	lightChannelMatrix.append(lightChannelMatrix, row1)
	lightChannelMatrix.append(lightChannelMatrix, row2)
	lightChannelMatrix.append(lightChannelMatrix, row3)

	row1 = {channels.BUTTON_UP1 channels.BUTTON_DOWN1, channels.BUTTON_COMMAND1}
	row2 = {channels.BUTTON_UP2 channels.BUTTON_DOWN2, channels.BUTTON_COMMAND2}
	row3 = {channels.BUTTON_UP3 channels.BUTTON_DOWN3, channels.BUTTON_COMMAND3}
	buttonChannelMatrix.append(buttonChannelMatrix, row1)
	buttonChannelMatrix.append(buttonChannelMatrix, row2)
	buttonChannelMatrix.append(buttonChannelMatrix, row3)
	for f = 0; f < typedef.N_FLOORS; f++ {
		for b = ButtonType.UP; b < typedef.N_BUTTONS; b++ {
			setButtonLight(f, b, 0)
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
				if checkButtonPressed(buttonType, floor){
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
		for floor := 0; floor < typedef.N_FLOORS; floor++ {
			floor := checkFloor(floor);
			if floor != CurrentFloor {
				PreviousFloor = CurrentFloor
				CurrentFloor = floor
				if  floor > -1 {
					floorChannel <- floor
				}
			}
			
		}
	}
}
// This function runs continously as a goroutine, waiting for orders to set lights.
func setLights(lightChannel <-chan typedef.LightEvent) {
	select{
		case lightEvent <- lightChannel:
			switch lightEvent.LightType {
			case typedef.BUTTON_CALL_UP:
				setButtonLight(lightEvent.Floor, 0, lightEvent.Value)
			case typedef.BUTTON_CALL_DOWN:
				setButtonLight(lightEvent.Floor, 1, lightEvent.Value)
			case typedef.BUTTON_COMMAND:
				setButtonLight(lightEvent.Floor, 2, lightEvent.Value)
			case INDICATOR_FLOOR:
				setFloorIndicator(lightEvent.Floor)
			case typedef.BUTTON_STOP:
				setStopLamp(lightEvent.value)
			case typedef.DOOR_LAMP:
				setDoorLamp(lightEvent.Value)
			default:
				// Do some error handling.
			}
	}
}

// This function runs continously as a goroutine, waiting for orders to set the motor direction
func motorControl(motorChannel <-chan typedef.MotorEvent) {
	setMotorDirection(<-motorChannel.Direction)
}
/*
	This functions loops through the different types of buttons at all the
	floors and checks if any buttons are pressed.
*/
func checkButtonPressed(buttonType, floor int) (pressed int){
	// TODO -> Do this better in terms of counter variable names and button types.
	if IOReadBit(buttonChannelMatrix[floor][buttonType]){
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
	if IOReadBit(channels.SENSOR_FLOOR1) {
		return 0
	} else if IOReadBit(channels.SENSOR_FLOOR2) {
		return 1
	} else if IOReadBit(channels.SENSOR_FLOOR3) {
		return 2
	} else if IOReadBit(channels.SENSOR_FLOOR4) {
		return 3
	} else {
		return -1
	}
}

/*
	This function checks the status of the stop button
*/
func checkStopSignal() int {
	return IOReadBit(channels.STOP)
}

/*
	This function checks the status of the obstruction button/signal
*/
func checkObstructionSignal() int {
	return IOReadBit(channels.OBSTRUCTION)
}

/*
	This function/channel (called from another goroutine) sets the light of a 
	specific type at the given floor to the specified value.
*/
func setButtonLight(floor, buttonType, value int) error{
	// TODO -> Some acceptance test for the arguments..
	if value {
		IOSetBit(lightChannelMatrix[floor][buttonType])
	} else {
		IOSetBit(lightChannelMatrix[floor][buttonType])
	}
}

/*
	This function/channel(called from another goroutine) sets the direction of
	the motor(any other direction than 0/STOP means it will run in this direction
	immediately).
*/
func setMotorDirection(direction Direction) error {
	if direction == 0 {
		IOWriteAnalog(channels.MOTOR, 0)
	} else if direction > 0 {
		IOClearBit(channels.MOTORDIR)
		IOWriteAnalog(channels.MOTOR, motorspeed)
	} else if direction < 0 {
		IOClearBit(channels.MOTORDIR)
		IOWriteAnalog(channels.MOTOR, motorspeed)
	}
	PreviousDirection = CurrentDirection
	CurrentDirection = direction

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
	if floor & 0x02 {
		IOSetBit(channels.LIGHT_FLOOR_IND1)
	} else {
		IOClearBit(channels.LIGHT_FLOOR_IND1)
	}

	if floor & 0x01 {
		IOSetBit(channels.LIGHT_FLOOR_IND2)
	} else {
		IOSetBit(channels.LIGHT_FLOOR_IND2)
	}
}

/*
	This function sets the value of the door lamp
*/	
func setDoorLamp(value int) {
	if value {
		IOSetBit(channels.LIGHT_DOOR_OPEN)
	} else {
		IOClearBit(channels.LIGHT_DOOR_OPEN)
	}
}

/*
	This function sets the value of the stop lamp.
*/
func setStopLamp(int value) {
	if value {
		IOSetBit(LIGHT_STOP)
	} else {
		IOClearBit(LIGHT_STOP)
	}
}















