package Hardware
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
	#import driver/elev.h
*/
import "C"
import "unsafe"

/*
	This type defines the different types of orders available in the system.
	The floor associated with the order is the floor the order is booked at.
*/

//		ALL OF THESE TYPES REQUIRE THE TYPES CAN BE REFERENCED FROM ANOTHER MODULE!
//		--------------------------------------------------------------------
type Order int const(
	NoOrder Order = iota
	InternalOrder // Order from within the elevator.
	orderUp // Order from outside the elevator, in the up direction.
	orderDown // Order from outside the elevator, in the down direction.
	floorIndicator // The floor indicators, that indicate when we are at a given floor.
)

/*
	This type defines the different types of lights available on the elevator
	control panel.
*/
type Light int const(
	Stop
	Up
	Down
	Internal
	floorIndicator
)

type Direction int const(
	Down = iota // TODO -> NEED TO SET THIS TO START AT -1 !!
	Stop
	Up
)

//		--------------------------------------------------------------------

var numberOfFloors int = 4; // This variable is set by the main loop. Default = 4


func hardwareLoop(){
	
	// This while loop runs continously, pinging the hardware.
	while(1){
		// Check if there are any new orders(buttons pressed).
		orderType, floor = checkButtons()
		if(orderType == orderUp){
			// TODO -> Do callback to master's main module with floor & up dir.
		}
		else if(orderType == orderDown){
			// TODO -> Do callback to master's main module with floor & down dir.
		}else if(orderType == internalOrder){
			// TODO -> Add the order to this elevators que.
		}
		// If there isn't any order, we don't do anyting.
		
		// Check if the elevator is inbetween or at a floor, -1 => between.
		floor = checkFloor()
		if(floor != -1){
			// TODO -> Check if the elevator has any orders at this floor,
			// if so -> stop.
		}
		// If we don't have any orders at this floor, we continue in the same
		// direction.
	}
}

/*
	This functions loops through the different types of buttons at all the
	floors and checks if any buttons are pressed.
*/
func checkButtons() (floor int, orderType Order){
	// TODO -> Do this better in terms of counter variable names and button types.
	for floor := 0; floor < numberOfFloors; floor ++ {
		for buttonType := 0; buttonType < numberOfButtonTypes; buttonType ++ {
			// TODO -> Call hardware function with floor and buttonType to 
			//			check if button is pressed.
			pressed, floor, orderType := // Some hardware function.
			if(pressed){
				// TODO -> Callback to Main module, this button at this floor is pressed.
				
				// TODO -> Should we return at this point or continue to iterate?
				// 			I support the latter, as this will detect multiple
				//			buttons being pressed at "once".
			}
		}
	}
}

/*
	This functions loops through all the floors to check the sensor at a
	given floor to see if the elevator is at that floor.
*/
func checkFloor() (floor int){
	for floor := 0; floor < numberOfFloors; floor ++ {
		// TODO -> Check the floor sensor at this floor to see if the elevator
		//			is at this floor.
		floor := // Some hardware function, -1 if not as this floor.
		if(floor != 0){
			// TODO -> Callback to Main module, that the elevator is at this floor.
			
			// TODO -> In this case i think we should return as the elevator
			//			cannot be at multiple floors at the same time.
		}
	}
}

/*
	This function/channel (called from another goroutine) sets the light of a 
	specific type at the given floor to the specified value.
*/
func setLight(floor, value int, lightType Light) (int errorCode){
	if(lightType == Stop){
		C.elev_set_stop_lamp(value);
		// TODO -> Do some acceptance test to see if the light were set.
	}
	else if(lightType == (Up || Down || Internal)){
		C.elev_set_button_lamp(lightType, floor, value);
		// TODO -> Do some acceptance test to see if the light were set.
	}
	else if(lightType == floorIndicator){
		// Call the C function from the elev.c file.
		C.elev_set_floor_indicator(floor);
		// TODO -> Do some acceptance test to see if the light were set.
	}	
	return -1
}

/*
	This function/channel(called from another goroutine) sets the direction of
	the motor(any other direction than 0/STOP means it will run in this direction
	immediately).
*/
func setMotorDirection(direction Direction) (int errorCode){
	C.elev_set_motor_direction(direction)
	// TODO -> Do some acceptance test to see if the direction was set.
	
	// If acceptance test completes.
	return 1
}