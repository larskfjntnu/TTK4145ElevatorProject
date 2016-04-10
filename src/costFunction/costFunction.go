package costFunction

/*
	This module is used to calculate wich elevator should respond to a given
	order. This is run as a separate goroutine, which does not run continously,
	but is started by the Master Elevator's Main module. It runs through
	the compution of the cost function, then terminates.
*/

	import (
	."../typedef"
	"fmt"
	)

/*
	This is the function that is run as a goroutine(or not?). It takes the the floor
	the elevator is being ordered to and the direction the "customer" wants
	to go as arguments and calculates the optimal (at least we hope it does)
	elevator to respond to the call.
*/
func calculateRespondingElevator(knownElevators map[string]*Elevator, activeElevators [string]bool, orderType, floor int) (assignedTo string, floor, type int) {
	if orderType != (BUTTON_CALL_DOWN || BUTTON_CALL_UP){
		return "", fmt.Errorf("Invalid ordeType")
	}
	if floor < 0 || floor > N_FLOOR - 1{
		return "", fmt.ErrorF("Invalid floor")
	}
	// TODO -> Implement some algorithm to calculate the optimal elevator
	lowestCost := 100000
	respondingElevator := 0
	for IP, active := range(activeElevators){

		if active{
			queue = knownElevators[IP].State.ExternalOrders // Gets a copy of the queue matrix for the current active elevator
			prevFloor := knownElevators[IP].State.PrevFloor
			direction := knownElevators[IP].State.CurrentDirection
			moving := knownElevators[IP].IsMoving()

			var orderDir int

			//prevFloor, currentFloor, direction = knownElevators[activeElevators[n,2]] 

			cost := 0
			//elevatorIp = activeElevators[n]
			// Tar ikke med det caset der heisen er allerede i beordret etasje i tilfelle vi har en annen ordning for det
			if moving { // elevator between floors and a shortert travel to next floor
				cost++
			}
			else{
				cost += 2
			}
			currentFloor := prevFloor
			for m := 0, m < 2*N_FLOOR, m++{
				nextFloor := knownElevators[IP].GetNextDirection(direction, currentFloor)
				cost += 2
				if direction == orderType & currentFloor == floor{//Dette er etasjen som er bestilt
					 if cost < lowestCost{
					 	lowestCost = cost
					 	respondingElevator = IP
					 	break
					 }
				}
				else if shouldStop(direction, currentFloor){// går ut fra at vi lager en slik funksjon, dette er for ordre som allerede eksisterer for heisen
					queue[floor][internal] = 0
					queue[floor][direction] = 0
					cost += 2
				}
			}
		}
		if respondingElevator!= 0{
			return respondingElevator, floor, buttonType
		}
	}
}

func newFloor(direction, currentFloor) int{
	//calculate new floor and direction according to the current direction, floor and the queue

}
