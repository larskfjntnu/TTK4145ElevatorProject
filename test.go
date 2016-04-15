package main

import(
	"./src/costFunction1"
	."./src/typedef"
	"fmt"
	"log"
	"time"
)

func main(){
	knownElevators := make(map[string]*Elevator)
	elev1 := Elevator{State: StateStruct{PrevFloor: 3,
										CurrentDirection: DIR_DOWN,
										PrevDirection: DIR_UP,
					},}
	elev1.State.ExternalOrders = [2][4]bool{{false, false, false, false},{true, false, false, false}}
	elev1.Time = (time.Now()).Add(-10*time.Millisecond)
	fmt.Println(elev1.MakeQueue())

	elev2 := Elevator{State: StateStruct{PrevFloor: 3,
										CurrentDirection: DIR_DOWN,
										PrevDirection: DIR_UP,
					},}
	elev2.State.ExternalOrders = [2][4]bool{{false, true, true, false},{false, true, false, false}}
	elev2.Time = (time.Now()).Add(-10*time.Millisecond)
	fmt.Println(elev2.MakeQueue())

	elev3 := Elevator{State: StateStruct{PrevFloor: 3,
										CurrentDirection: DIR_DOWN,
										PrevDirection: DIR_UP,
					},}
	elev3.State.ExternalOrders = [2][4]bool{{false, true, false, true},{true, true, true, false}}
	elev3.Time = (time.Now()).Add(-2*time.Second)
	fmt.Println(elev3.MakeQueue())

	fmt.Println("\n\nReassigning orders\n\n")

	knownElevators["1"] = &elev1
	knownElevators["2"] = &elev2
	knownElevators["3"] = &elev3
	active := make(map[string]bool)
	active["1"] = false
	active["2"] = true
	active["3"] = true

	checkOnlineAndReassign(knownElevators, active, "1", 1*time.Second)
	fmt.Println((time.Since(elev3.Time) >= 1*time.Second))
}


func checkOnlineAndReassign(knownElevators map[string]*Elevator, activeElevators map[string]bool, localIP string, timeoutLimit time.Duration) {
	
	removed := make(map[string]*Elevator)

	for key := range knownElevators{
		fmt.Println(key)
		if key == localIP{
			activeElevators[key] = false // dont want to see ourself
		}else if time.Since(knownElevators[key].Time) <= timeoutLimit {
			fmt.Println(time.Since(knownElevators[key].Time))
			if activeElevators[key] != true {
				activeElevators[key] = true
				fmt.Println("Added active elevator " + knownElevators[key].State.LocalIP)
			}
		} else {
			if activeElevators[key] == true {
				fmt.Println("Removing inactive elevator " + knownElevators[key].State.LocalIP)
				removed[key] = knownElevators[key]
				activeElevators[key] = false
			}
		}
	}

	for _, elevator := range(removed){
		exDown := elevator.State.ExternalOrders[0]
		exUp := elevator.State.ExternalOrders[1]

		for floor, ordered := range exDown{
			if ordered{
				assigned, err := costFunction1.CalculateRespondingElevator(knownElevators, activeElevators, localIP, BUTTON_CALL_DOWN, floor)
				if err != nil{
					log.Fatal(err)
				} else {
					fmt.Printf("Order %s at floor %d assigned to %s\n", HardwareEventType[BUTTON_CALL_DOWN], floor, assigned)
				}
			}
		}

		for floor, ordered := range exUp{
			if ordered{
				assigned, err := costFunction1.CalculateRespondingElevator(knownElevators, activeElevators, localIP, BUTTON_CALL_UP, floor)
				if err != nil{
					log.Fatal(err)
				} else {
					fmt.Printf("Order %s at floor %d assigned to %s\n", HardwareEventType[BUTTON_CALL_UP], floor, assigned)
				}
			}
		}
	}
}