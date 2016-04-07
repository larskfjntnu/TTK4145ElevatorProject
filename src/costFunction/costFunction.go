package CostFunction

/*
	This module is used to calculate wich elevator should respond to a given
	order. This is run as a separate goroutine, which does not run continously,
	but is started by the Master Elevator's Main module. It runs through
	the compution of the cost function, then terminates.
*/

/*
	This is the function that is run as a goroutine(or not?). It takes the the floor
	the elevator is being ordered to and the direction the "customer" wants
	to go as arguments and calculates the optimal (at least we hope it does)
	elevator to respond to the call.
*/
func calculateRespondingElevator(knownElevators map[string]*Elevator, activeElevators []string) (assignedTo string, floor, type int) {
	// TODO -> Implement some algorithm to calculate the optimal elevator

}
