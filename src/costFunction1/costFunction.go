package costFunction1
import(
"fmt"
."../typedef"
)
func CalculateRespondingElevator(knownElevators map[string]*Elevator, activeElevators map[string]bool, localIP string, orderType, floor int) (assignedTo string, err error){
	if !(orderType >= 0 && orderType <= 1){
		return "", fmt.Errorf("Invalid orderType")
	}
	if floor < 0 || floor > N_FLOORS-1 {
		return "", fmt.Errorf("Invalid floor")
	}

	// To compare diretions with order
	bdir := 0
	if orderType == BUTTON_CALL_UP{
		bdir = DIR_UP
	} else {
		bdir = DIR_DOWN
	}

	lowestCost := 1000000
	respondingElevator := ""
	for IP, active := range(activeElevators){
		if active || (IP == localIP) {
			tempElevatorObj := *knownElevators[IP] // Make a copy of the elevator
			tempElevator := &tempElevatorObj


			if tempElevator.State.ExternalOrders[orderType][floor]{
				// Already have this order
				return IP, nil
			}

			if tempElevator.GetFloor() == floor && !tempElevator.IsMoving(){
				// Standstill at the ordered floor
				if tempElevator.GetNextDirection() == DIR_DOWN && orderType == BUTTON_CALL_DOWN{
					// And going in the right direction
					return IP, nil
				}
				if tempElevator.GetNextDirection() == DIR_UP && orderType == BUTTON_CALL_UP{
					return IP, nil
				}
				if tempElevator.GetNextDirection()== DIR_STOP {
					// Or going nowhere
					return IP, nil
				}
			}

			// Simulate route to the new order
			tempElevator.State.ExternalOrders[orderType][floor] = true
			cost := 0

			if tempElevator.IsMoving(){
				cost ++
			} else {
				cost += 2
			}



			costloop:
			for m:= 0; m < 3*N_FLOORS; m++{ // Should be 2*N_FLOORS at most, but give it some slack
				// Set next floor and direction
				cost += 2
				//f := tempElevator.GetFloor()
				tempElevator.SetDirection(tempElevator.GetNextDirection())
				tempElevator.SetFloor(tempElevator.GetFloor() + tempElevator.GetDirection())
				fmt.Printf("Dir: 	%s, toFloor:	%d\n", MotorDirections[tempElevator.GetDirection()+1], tempElevator.GetFloor())
				//fmt.Printf("From floor %d to %d\n", f, tempElevator.GetFloor())
				//fmt.Printf("Last direction %s, Current direction %s\n", MotorDirections[tempElevator.State.PrevDirection + 1], MotorDirections[tempElevator.GetDirection() + 1])

				if tempElevator.ShouldStop(){
					tempElevator.SetDirection(DIR_STOP)
					fmt.Println("Stopped")
					if tempElevator.GetFloor() == floor{
						// We are at the ordered floor
						if (tempElevator.GetNextDirection() == bdir) || (tempElevator.GetNextDirection() == DIR_STOP){
							// And going in the right direction.
							if cost < lowestCost{
								lowestCost = cost
								respondingElevator = IP
								break costloop
							}

						}
					}
					// Cancel correct orders at this floor
					d := 0
					if tempElevator.GetDirection() == DIR_UP || tempElevator.GetNextDirection() == DIR_UP{
						d = BUTTON_CALL_UP
					}else if tempElevator.GetDirection() == DIR_DOWN || tempElevator.GetNextDirection() == DIR_DOWN{
						d = BUTTON_CALL_DOWN
					}
					fmt.Printf("Cancel order %d at floor %d\n", d, tempElevator.GetFloor())
					fmt.Println(tempElevator.MakeQueue())
					tempElevator.SetInternalOrder(tempElevator.GetFloor(), false)
					tempElevator.State.ExternalOrders[d][tempElevator.GetFloor()] = false
					if !tempElevator.State.OrdersAbove() || !tempElevator.State.OrdersBelow(){
						tempElevator.State.ExternalOrders[0][tempElevator.GetFloor()] = false
						tempElevator.State.ExternalOrders[1][tempElevator.GetFloor()] = false
					}


					//fmt.Println(tempElevator.MakeQueue())
					fmt.Println(tempElevator.MakeQueue())
				}
				cost += 2
			}
		}
	}
	if respondingElevator == ""{
		err = fmt.Errorf("Costfunction could not assign order to elevator")
	}else{
		err = nil
	}
	return respondingElevator, err
}