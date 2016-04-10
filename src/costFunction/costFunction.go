package costFunction

/*
	This module is used to calculate wich elevator should respond to a given
	order.
*/

/*
	This is the function that calculates the 'optimal' elevator. It takes the the floor
	the elevator is being ordered to and the direction the "customer" wants
	to go as arguments and calculates the optimal elevator to respond to the call.
	It also needs all elevators states which are found by using the activeElevators
	to access knownElevators
*/
func calculateRespondingElevator(knownElevators map[string]*Elevator, activeElevators []string) (assignedTo string, floor, type int) {
	// TODO -> Implement some algorithm to calculate the optimal elevator

	

}