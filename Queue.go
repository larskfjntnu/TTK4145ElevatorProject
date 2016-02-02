/*
	This module keeps this elevators queue.
	The queue is implemented as a m x 3 matrix
	(m length array which contains arrays of length 3) where 
	each row represent the three possible orders (up, down, internal) at
	each of the m floors.
	Implements functions to set orders and to check orders.
	This module does not run as a goroutine, but keeps track of the queue
	and is queried whenever needed.
	It also implements an initialize function, which allocates the queue.
*/

var [][] queue;

/*
	This function initalizes the queue.
*/

func initialize(int numberOfFloors, numberOfOrderTypes) (int errorCode){
	queue := make([][]int, numberOfOrderFloors)
	for row := 0; row < numberOfFloors; row++ {
		queue[row] = make([]int, numberOfTypes)
		for element := range queue[row]{
			queue[row][element] = 0;
		}
	}
	
	// TODO -> AcceptanceTest, check the dimensions of the array
	//			and also possible that all elemets sum to zero.
}

/*
	This function sets or cancels an order at the given floor, and
	of the given type based on the value.
*/
func setOrder(floor, orderType, value int){
	queue[floor][orderType] = value;
	// TODO -> AcceptanceTest, check the element is what we set it to.
}

/*
	This function checks if the floor is "ordered" in the given direction.
	If error, elevatorIsOrdered = -1;
*/
func checkOrder(floor, direction int) (int elevatorIsOrdered){
	// TODO -> Do some logic to convert direction to column index
	// Check if ordered internally
	if(queue[floor][3] == 1){ // This is also an acceptance test, why?
		// TODO -> The floor is ordered internally, do callback to main module.
	}
	else if(queue[floor][direction] == 0){ // This is also an acceptance test, why?
		// TODO -> The floor is ordered, do callback to main module
	}
}
