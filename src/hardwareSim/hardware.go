package hardwareSim
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
#cgo LDFLAGS: -lpthread -lcomedi -lcomedi -lm
#cgo CFLAGS: -std=gnu11
#include "elev.h"
*/
import "C"
import "fmt"
import "time"
import "log"
import "strconv"
import ."../typedef"


// ------------------------- CONSTANT and VARIABLE DECLERATIONS
var lightChannelMatrix = [N_FLOORS][N_BUTTONS]int {
	{LIGHT_DOWN1, LIGHT_UP1, LIGHT_COMMAND1},
	{LIGHT_DOWN2, LIGHT_UP2, LIGHT_COMMAND2},
	{LIGHT_DOWN3, LIGHT_UP3, LIGHT_COMMAND3},
	{LIGHT_DOWN4, LIGHT_UP4, LIGHT_COMMAND4},
}
var buttonChannelMatrix = [N_FLOORS][N_BUTTONS]int {
	{BUTTON_DOWN1, BUTTON_UP1, BUTTON_COMMAND1},
	{BUTTON_DOWN2, BUTTON_UP2, BUTTON_COMMAND2},
	{BUTTON_DOWN3, BUTTON_UP3, BUTTON_COMMAND3},
	{BUTTON_DOWN4, BUTTON_UP4, BUTTON_COMMAND4},
}

type ButtonEvent struct{
	Type int
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

type FloorSensorEvent struct{
	CurrentDirection int
	Floor int
}

var PreviousFloor int
var CurrentFloor int
var CurrentDirection int
var PreviousDirection int
var initialized bool = false
const motorspeed = 2800


var debug bool = false


//		-----------------------  FUNCTION DECLERATIONS    -----------------------------------


func Init(hardwareEventChannel chan HardwareEvent ,delayInPolling time.Duration) error{
	if initialized{
		return fmt.Errorf("Hardware is already initialized.")
	}
	C.elev_init(1)
	resetLights()
	SetMotorDirection(DIR_STOP)
	
	// If initialized between floors, move down to nearest floor.
	startingFloor := -1
	if startingFloor = checkFloor(); startingFloor == -1 {
		printDebug("Starting between floors, going down")
		SetMotorDirection(DIR_DOWN)
		for {
			if startingFloor = checkFloor(); startingFloor != -1 {
				printDebug("INIT -> Arrived at floor: " + strconv.Itoa(startingFloor))
				SetMotorDirection(DIR_STOP)
				hardwareEventChannel <- HardwareEvent{Event: EventFloorReached,
														CurrentDirection: DIR_DOWN,
														Floor: startingFloor}
				setFloorIndicator(startingFloor)
				break
			} else {
				time.Sleep(delayInPolling)
			}
		}
	} else {
		hardwareEventChannel <- HardwareEvent{Event: EventFloorReached, CurrentDirection: DIR_STOP, Floor: startingFloor}
		setFloorIndicator(startingFloor)
	}

	// Start goroutines to handle polling hardware
	go hardwareRoutine(hardwareEventChannel, delayInPolling, startingFloor)
	return nil
}

/*  This function runs continously as a goroutine, handling two way commmunication with the
	main loop.
*/
func hardwareRoutine(hardwareEventChannel chan HardwareEvent, delayInPolling time.Duration, startingFloor int){
	buttonChannel := make(chan ButtonEvent)
	floorSensorChannel :=make(chan FloorSensorEvent)
	
	go buttonPolling(buttonChannel, delayInPolling)
	go floorSensorPolling(floorSensorChannel, delayInPolling, startingFloor)
	for{
		select{
			case btEvent := <- buttonChannel:
				hardwareEventChannel <- HardwareEvent{ Event: EventButtonPressed,
														Floor: btEvent.Floor, 
														ButtonType: btEvent.Type,
														}
			case fSEvent := <- floorSensorChannel:
				hardwareEventChannel <- HardwareEvent{ Event: EventFloorReached,
														Floor: fSEvent.Floor,
														CurrentDirection: fSEvent.CurrentDirection,
														}
		}
	}
}


// This function runs continously as a goroutine, pinging the hardware for button presses.
func buttonPolling(buttonChannel chan ButtonEvent, delayInPolling time.Duration){
	readingMatrix := [N_FLOORS][N_BUTTONS]bool{}
	stopButton := false
	stopState  := false
	obstructionSignal := false

	// This while loop runs continously, polling the hardware for button presses.
	for {
		// Check if there are any new orders(buttons pressed).
		for floor := 0; floor < N_FLOORS; floor ++ {
			for buttonType := BUTTON_CALL_DOWN; buttonType < BUTTON_COMMAND + 1; buttonType++ {
				if checkButtonPressed(buttonType, floor) {
					if !readingMatrix[floor][buttonType] {
						readingMatrix[floor][buttonType] = true
						// Pass a hardwareevent to the event channel.
						buttonChannel <- ButtonEvent{Type: buttonType, Floor: floor}
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
				if stopButton && !stopState{
					// First time we press stop
					buttonChannel <- ButtonEvent{Type: BUTTON_STOP, Value: true}
					stopState=true
				} else if stopButton &&stopState{
					// Second time we press stop
					buttonChannel <- ButtonEvent{Type: BUTTON_STOP, Value: false}
					stopState=false
				}

				
			} else{
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
		time.Sleep(delayInPolling)
	}
}

// This function runs continously as a goroutine, pinging the hardware for floor arrivals.
func floorSensorPolling(floorChannel chan FloorSensorEvent, delayInPolling time.Duration, startingFloor int){
	lastFloor := startingFloor
	for{
		floor := checkFloor()
		if (floor != -1) && (floor != lastFloor){
			dir := 0
			if lastFloor > floor{
				dir = -1
			} else if lastFloor < floor{
				dir = 1
			}
			lastFloor = floor
			setFloorIndicator(floor)
			floorChannel <- FloorSensorEvent{Floor: floor, CurrentDirection: dir} 
			}	
		time.Sleep(delayInPolling)
	}
}

// --------------------- Check hardware functions ----------------------------
/*
	This functions loops through the different types of buttons at all the
	floors and checks if any buttons are pressed.
*/
func checkButtonPressed(buttonType, floor int) bool {
	// TODO -> Do this better in terms of counter variable names and button types. Can this function be removed?
	var typ C.elev_button_type_t
	if buttonType == BUTTON_CALL_UP{
		typ = C.BUTTON_CALL_UP
	} else if buttonType == BUTTON_CALL_DOWN{
		typ = C.BUTTON_CALL_DOWN
	} else if buttonType == BUTTON_COMMAND{
		typ = C.BUTTON_COMMAND
	}



	if C.elev_get_button_signal(typ, C.int(floor)) != 0{
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
	return int(C.elev_get_floor_sensor_signal())
}

/*
	This function checks the status of the stop button
*/
func checkStopSignal() bool {
	return (C.elev_get_stop_signal() != 0)
}

/*
	This function checks the status of the obstruction button/signal
*/
func checkObstructionSignal() bool {
	return (C.elev_get_obstruction_signal() != 0)
}


// ------------------------ Set Functions -----------------------------

/*
	This function/channel(called from another goroutine) sets the direction of
	the motor(any other direction than 0/STOP means it will run in this direction
	immediately).
*/
func SetMotorDirection(direction int) error {
	printDebug("Setting motor direction: " + strconv.Itoa(direction))
	var dir C.elev_motor_direction_t
	if direction == DIR_UP {
		dir = C.DIRN_UP
	} else if direction == DIR_DOWN {
		dir = C.DIRN_DOWN
	} else if direction == DIR_STOP{
		dir = C.DIRN_STOP
	}

	C.elev_set_motor_direction(dir)

	// TODO -> Do some acceptance test to see if the direction was set.
	// return some error.New()..
	// If acceptance test completes.
	return nil
}

func cbool(b bool) C.int{
	if b{
		return C.int(1)
	} else{
		return C.int(0)
	}
}

// This function sets the lights based on a LightEvent.
func SetLights(lightType, floor int, value bool){
	switch lightType{
	case BUTTON_CALL_UP, BUTTON_CALL_DOWN, BUTTON_COMMAND:
		var typ C.elev_button_type_t
		if lightType == BUTTON_CALL_UP{
			typ = C.BUTTON_CALL_UP
		} else if lightType == BUTTON_CALL_DOWN{
			typ = C.BUTTON_CALL_DOWN
		} else if lightType == BUTTON_COMMAND{
			typ = C.BUTTON_COMMAND
		}

		C.elev_set_button_lamp(typ, C.int(floor), cbool(value))
	case BUTTON_STOP:
		C.elev_set_stop_lamp(cbool(value))
	case DOOR_LAMP:
		C.elev_set_door_open_lamp(cbool(value))
	default:
		// Do some error handling.
	}
}


/*
	This function/channel (called from another goroutine) sets the light of a 
	specific type at the given floor to the specified value.

func setButtonLight(floor, buttonType int, value bool) error {
	// TODO -> Some acceptance test for the arguments..
	if value {
		driver.IOSetBit(lightChannelMatrix[floor][buttonType])
	} else {
		driver.IOClearBit(lightChannelMatrix[floor][buttonType])
	}
	return nil
}
/*

/*
	This function sets the indicator at a given floor.
*/
func setFloorIndicator(floor int) {
	// Binary encoding, one light is always on 00, 01, 10 or 11
	if floor >= N_FLOORS || floor < 0 {
		log.Println("HARDWARE:\t Tried to set indicator on invalid floor.")
		// todo set floor to nearest valid floor.
	}
	C.elev_set_floor_indicator(C.int(floor))
}

/*
	This function sets the value of the door lamp
*/	
func setDoorLamp(value bool) {
	C.elev_set_door_open_lamp(cbool(value))
}

/*
	This function sets the value of the stop lamp.

func setStopLamp(value bool) {
	if value {
		driver.IOSetBit(LIGHT_STOP)
	} else {
		driver.IOClearBit(LIGHT_STOP)
	}
}

*/


func resetLights() {
	for f:=0;f< N_FLOORS;f++{
		for b:= BUTTON_CALL_DOWN;b<N_BUTTONS;b++{
			var typ C.elev_button_type_t
			if b == BUTTON_CALL_UP{
				typ = C.BUTTON_CALL_UP
			} else if b == BUTTON_CALL_DOWN{
				typ = C.BUTTON_CALL_DOWN
			} else if b == BUTTON_COMMAND{
				typ = C.BUTTON_COMMAND
			}
			C.elev_set_button_lamp(typ, C.int(f), cbool(false))
		}
	}
	C.elev_set_stop_lamp(C.int(0))
	C.elev_set_door_open_lamp(C.int(0))
}

// ----------------  Temporary functions to reset elevator from separate program. ----------------

func printDebug(message string){
	if debug{
		log.Println("Hardware:\t" + message)
	}
}
