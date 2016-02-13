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
	"channels"
	"fmt"
	"time"
	)


// ------------------------- CONSTANT and VARIABLE DECLERATIONS
var lightChannelMatrix = [N_FLOORS][N_BUTTONS]{
	{LIGHT_UP1, LIGHT_DOWN1, LIGHT_COMMAND1},
	{LIGHT_UP2, LIGHT_DOWN2, LIGHT_COMMAND2},
	{LIGHT_UP3, LIGHT_DOWN3, LIGHT_COMMAND3},
	{LIGHT_UP4, LIGHT_DOWN4, LIGHT_COMMAND4},
}
var buttonChannelMatrix = [N_FLOORS][N_BUTTONS]{
	{BUTTON_UP1, BUTTON_DOWN1, BUTTON_COMMAND1},
	{BUTTON_UP2, BUTTON_DOWN2, BUTTON_COMMAND2},
	{BUTTON_UP3, BUTTON_DOWN3, BUTTON_COMMAND3},
	{BUTTON_UP4, BUTTON_DOWN4, BUTTON_COMMAND4},
}

type ButtonEvent struct{
	ButtonType int
	Floor int
	Value bool
}

type LightEvent struct{
	LightType int
	Floor int
	Value bool
}

type MotorEvent struct{
	MotorDirection int
}

type FloorEvent struct{
	CurrentDirection Direction
	Floor int
}

var PreviousFloor int
var CurrentFloor int
var CurrentDirection int
var PreviousDirection int
var initialized bool = false
const motorspeed = 2800


//		-----------------------  FUNCTION DECLERATIONS    -----------------------------------


func Init(buttonChannel chan<- ButtonEvent, lightChannel <-chan LightEvent, motorChannel <-chan MotorEvent, floorChannel chan<- FloorEvent, pollingDelay time.Duration) error{
	if initialized{
		return fmt.Errorf("Hardware is already initialized.")
	}
	initSuccess := IOInit()
	if initSuccess!=nil{
		return fmt.Errorf("Unable to initialize hardware.")
	}
	resetLights()

	// Start goroutines to handle hardware events.
	go controlLights(lightChannel)
	go controlMotor(motorChannel)
	motorChannel<-DIR_STOP
	// If initialized between floors, move down to nearest floor.
	if checkFloor() == -1 {
		motorChannel <- DIR_DOWN
		for {
			if checkFloor() != -1 {
				motorChannel <- DIR_STOP
				break
			} else {
				time.Sleep(pollingDelay)
			}
		}
	}
	go readButtons(buttonChannel, pollingDelay)
	go readFloorSensors(floorChannel, pollingDelay)
	return nil
	// TODO -> Acceptance test!!!
}


// This function runs continously as a goroutine, pinging the hardware for button presses.
func readButtons(buttonChannel chan<- ButtonEvent, pollingDelay time.Duration){
	readingMatrix := [N_FLOORS][N_BUTTONS]bool{}
	var stopButton bool = false
	var stopState bool = false
	var obstructionSignal = false
	// This while loop runs continously, pinging the hardware for button presses.
	for {
		// Check if there are any new orders(buttons pressed).
		for floor := 0; floor < N_FLOORS; floor ++ {
			for buttonType := BUTTON_CALL_UP; buttonType < BUTTON_COMMAND; buttonType++ {
				if checkButtonPressed(buttonType, floor) {
					if !readingMatrix[floor][buttonType] {
						readingMatrix[floor][buttonType] = true
						// Pass a hardwareevent to the event channel.
						buttonChannel <- ButtonEvent{ButtonType: buttonType, Floor: floor}
					}
				} else {
					// Make sure readingMatrix is set to false for this button.
					// This makes sure that the buttonChannel is not filled with events.
					readingMatrix[floor][buttonType] = false
				}
			}
		}
		if checkStopSignal() {
			if !stopButton {
				stopButton = true
				if stopButton && !stopState {
					// First time we press stop
					buttonChannel <- ButtonEvent{ButtonType: BUTTON_STOP, Value: true}
					stopState = true
				}  if stopButton && stopState {
					// Second time we press stop
					buttonChannel <- ButtonEvent{ButtonType: BUTTON_STOP, Value: false}
					stopState  = false
				}

				
			} else {
				stopButton = false
			}
		}
		if checkObstructionSignal() {
			if !obstructionSignal {
				obstructionSignal = true
				
			} else {
				obstructionSignal = false
			}
		}
		time.Sleep(pollingDelay)
	}
}

// This function runs continously as a goroutine, pinging the hardware for floor arrivals.
func readFloorSensors(floorChannel chan<- FloorEvent, pollingDelay time.Duration) {
	var lastFloor := -1
	for {
		floor := checkFloor()
		if (floor != -1) && (floor != lastFloor) {
			lastFloor = floor
			setFloorIndicator(floor)
			floorChannel <- FloorEvent{Floor: floor} 
			}	
		}
		time.Sleep(pollingDelay)
	}
}
// This function runs continously as a goroutine, waiting for orders to set lights.
func controlLights(lightChannel <-chan LightEvent){
	select{
		case lightEvent:=<-lightChannel:
			switch lightEvent.LightType{
			case BUTTON_CALL_UP, BUTTON_CALL_DOWN, BUTTON_COMMAND:
				setButtonLight(lightEvent.Floor, lightEvent.LightType, lightEvent.Value)
			case BUTTON_STOP:
				setStopLamp(lightEvent.Value)
			case DOOR_LAMP:
				setDoorLamp(lightEvent.Value)
			default:
				// Do some error handling.
			}
	}
}

// This function runs continously as a goroutine, waiting for orders to set the motor direction
func controlMotor(motorChannel<-chan MotorEvent){
	motorEv :=<-motorChannel
	setMotorDirection(motorEvent.MotorDirection)
}


// --------------------- Check hardware functions ----------------------------
/*
	This functions loops through the different types of buttons at all the
	floors and checks if any buttons are pressed.
*/
func checkButtonPressed(buttonType, floor int) bool {
	// TODO -> Do this better in terms of counter variable names and button types. Can this function be removed?
	if IOReadBit(buttonChannelMatrix[floor][buttonType]){
		return true
	} else {
		return false
	}
}

/*
	This functions checks the sensor at a
	given floor to see if the elevator is at that floor.
*/
func checkFloor() int {
	if IOReadBit(SENSOR_FLOOR1) {
		return 0
	} else if IOReadBit(SENSOR_FLOOR2) {
		return 1
	} else if IOReadBit(SENSOR_FLOOR3) {
		return 2
	} else if IOReadBit(SENSOR_FLOOR4) {
		return 3
	} else {
		return -1
	}
}

/*
	This function checks the status of the stop button
*/
func checkStopSignal() bool {
	return IOReadBit(STOP)
}

/*
	This function checks the status of the obstruction button/signal
*/
func checkObstructionSignal() bool {
	return IOReadBit(OBSTRUCTION)
}


// ------------------------ Set motor -----------------------------

/*
	This function/channel(called from another goroutine) sets the direction of
	the motor(any other direction than 0/STOP means it will run in this direction
	immediately).
*/
func setMotorDirection(direction int) error {
	if direction == 0 {
		IOWriteAnalog(MOTOR, 0)
	} else if direction > 0 {
		IOClearBit(MOTORDIR)
		IOWriteAnalog(MOTOR, motorspeed)
	} else if direction < 0 {
		IOSetBit(MOTORDIR)
		IOWriteAnalog(MOTOR, motorspeed)
	}

	// TODO -> Do some acceptance test to see if the direction was set.
	// return some error.New()..
	// If acceptance test completes.
	return nil
}

// ------------------------ Light functions --------------------------------

/*
	This function/channel (called from another goroutine) sets the light of a 
	specific type at the given floor to the specified value.
*/
func setButtonLight(floor, buttonType int, value bool) error {
	// TODO -> Some acceptance test for the arguments..
	if value {
		IOSetBit(lightChannelMatrix[floor][buttonType])
	} else {
		IOClearBit(lightChannelMatrix[floor][buttonType])
	}
	return nil
}

/*
	This function sets the indicator at a given floor.
*/
func setFloorIndicator(floor int) {
	// Binary encoding, one light is always on 00, 01, 10 or 11
	if floor >= N_FLOORS || floor <= 0 {
		log.Println("HARDWARE:\t Tried to set indicator on invalid floor.")
		// todo set floor to nearest valid floor.
	}
	if bool((floor & 0x02) != 0) {
		IOSetBit(LIGHT_FLOOR_IND1)
	} else {
		IOClearBit(LIGHT_FLOOR_IND1)
	}
	if bool((floor & 0x01) != 0) {
		IOSetBit(LIGHT_FLOOR_IND2)
	} else {
		IOClearBit(LIGHT_FLOOR_IND2)
	}
}

/*
	This function sets the value of the door lamp
*/	
func setDoorLamp(value bool) {
	if value {
		IOSetBit(LIGHT_DOOR_OPEN)
	} else {
		IOClearBit(LIGHT_DOOR_OPEN)
	}
}

/*
	This function sets the value of the stop lamp.
*/
func setStopLamp(value bool) {
	if value {
		IOSetBit(LIGHT_STOP)
	} else {
		IOClearBit(LIGHT_STOP)
	}
}


func resetLights() {
	for f:=0;f<N_FLOORS;f++{
		for b:=BUTTON_CALL_UP;b<N_BUTTONS;b++{
			setButtonLight(f, b, false)
		}
	}
	setStopLamp(false)
	setDoorLamp(false)
}















