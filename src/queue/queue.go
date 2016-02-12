package queue

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

import "typedef"

var queue [][]int

/*
	This function initalizes the queue as a numberOfFloors x numberOfOrderTypes
	matrix.
*/

func Init(numberOfFloors, numberOfOrderTypes int) error {
	queue := make([][]int, numberOfOrderTypes)
	for row := 0; row < numberOfFloors; row++ {
		queue[row] = make([]int, numberOfOrderTypes)
		for element := range queue[row] {
			queue[row][element] = 0
		}
	}
	// TODO -> AcceptanceTest, check the dimensions of the array
	//			and also possible that all elemets sum to zero.
	return nil
}

/*
	This function sets or cancels an order at the given floor, and
	of the given type based on the value.
*/
func setOrder(floor, orderType, value int) {
	queue[floor][orderType] = value
	// TODO -> AcceptanceTest, check the element is what we set it to.
}

/*
	This function checks if the floor is "ordered" in the given direction.
	Down direction is 0, up is 1 ?
	If error, return = -1;
*/
func checkOrder(floor, direction int) int {
	// TODO -> Do some logic to convert direction to column index
	// Check if ordered internally
	if queue[floor][3] == 1 { // This is also an acceptance test, why?
		// TODO -> The floor is ordered internally
		return 1
	} else if queue[floor][direction] == 1 { // This is also an acceptance test, why?
		// TODO -> The floor is not ordered
		// 9/2 -16 changed if statetement from 0 to 1, possible typo?
		return 1
	}
	return 0
}

// Check if there are any orders. Returns -1 for down, 1 for up and 0 for none
func anyOrders(atFloor, previousDirection int) int {
	// TODO -> Also do this better.
	for floor := 0; floor < typedef.N_FLOORS; floor++ {
		for direction := 0; direction < 3; direction++ {
			if queue[floor][direction] != 0 {
				if atFloor > floor && previousDirection == 1 || atFloor < floor && previousDirection == -1 {
					return previousDirection
				} else if atFloor < floor {
					return -1
				} else if atFloor > floor {
					return 1
				}
			}
		}
	}
	return 0
}
