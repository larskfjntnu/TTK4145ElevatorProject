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
func CalculateRespondingElevator(knownElevators (map[string]*Elevator), activeElevators map[string]bool, orderType, floor int) (assignedTo string, err error) {

	if (orderType == BUTTON_CALL_DOWN) && (orderType == BUTTON_CALL_UP){
		return "", fmt.Errorf("Invalid orderType")
	}
	if (floor < 0) || (floor > N_FLOORS - 1) {
		return "", fmt.Errorf("Invalid floor")
	}

	lowestCost := 100000
	respondingElevator := ""
	for IP, active := range(activeElevators){
		if active{
			tempElevatorObj := *knownElevators[IP] // Gets a copy of the queue matrix for the current active elevator
			tempElevator := &tempElevatorObj

			if tempElevator.State.PrevFloor == floor && !tempElevator.State.HaveOrders(){
				return IP, nil
			} 

			if tempElevator.State.PrevFloor == floor && !tempElevator.IsMoving() && ((tempElevator.GetNextDirection() == DIR_UP && orderType == BUTTON_CALL_UP) || (tempElevator.GetNextDirection() == DIR_DOWN && orderType == BUTTON_CALL_DOWN)) {
				return IP, nil
			}

			if floor == 0 && tempElevator.State.PrevFloor == 0 && !tempElevator.IsMoving(){
				return IP, nil
			}

			if floor == N_FLOORS-1 && tempElevator.State.PrevFloor == N_FLOORS-1 && !tempElevator.IsMoving(){
				return IP, nil
			}



			// Insert new order for testing
			tempElevator.State.ExternalOrders[orderType][floor] = true
			cost := 0


			if tempElevator.IsMoving() { // Shorter travel to next floor
				cost++
			} else {
				cost += 2
			}
			costloop:
			for m := 0; m < 2*N_FLOORS; m++{

				dir := tempElevator.GetNextDirection()
				tempElevator.State.PrevFloor = tempElevator.State.PrevFloor + dir
				tempElevator.SetDirection(dir)
				cost += 2

				fmt.Println(tempElevator.State)
				if ((orderType == BUTTON_CALL_UP && dir == DIR_UP) || (orderType == BUTTON_CALL_DOWN && dir == DIR_DOWN)) && tempElevator.State.PrevFloor == floor{
					// We have arrived at the ordered floor
					 if cost < lowestCost{
					 	lowestCost = cost
					 	respondingElevator = IP
					 	break costloop
					 }
				} else if tempElevator.State.PrevFloor == floor && !tempElevator.State.HaveOrders() {
					// We have arrived at the ordered floor
					 if cost < lowestCost{
					 	lowestCost = cost
					 	respondingElevator = IP
					 	break costloop
					 }
				} else if tempElevator.ShouldStop(){
					tempElevator.State.InternalOrders[tempElevator.State.PrevFloor] = false
					i := 0
					if dir == DIR_DOWN{
						i = 1
					}
					tempElevator.State.ExternalOrders[i][tempElevator.State.PrevFloor] = false
					cost += 2
				}
			}
		}
	}
	if respondingElevator == ""{
		return "", fmt.Errorf("Error, no elevator found..")
	} else {
		return respondingElevator, nil
	}
}