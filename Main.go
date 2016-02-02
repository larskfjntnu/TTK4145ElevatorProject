/*
	This is the main module controlling the elevator and handling callbacks
	from other modules, as well as calling functions/channels from other
	modules/threads. This module interfaces with the Network and Hardware
	threads, as well as the Queue, CostFunction(if in MasterMode) and Debug
	functions. 
	The boolean masterMode keeps track of wether or not the elevator running
	this thread is a master or slave(OPT master/client) of the distributed
	system(several elevators running on a network).
	If the elevator is in master mode, it holds the responsability to 
	calculate which elevator should respond to a given order by using the 
	CostFunction module.
	If the elevator is not in master mode, it sends an external order to 
	the master and waits for the master to decide which elevator should 
	respond to the order.
*/

package main

// This function starts the program. The while loop inside runs throughout
// the lifetime of the program.
func main(){
	// Do some initialization. Determine wether Master or slave/client.
	
	// This is the main loop running continuously.
	while(1){
		// TODO -> Implement the main logic of the elevator.
	}
}

