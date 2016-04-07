package somepackage

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
import (
	"fmt"
	"time"
	"udp"
	"hardware"
	"time"
	"network"
	. "./src/typedef"
)

func main() {

	const delayInPolling = 50 * time.Milliseconds
	const attemptToConnectLimit = 5
	const elevOnlineTick = 100 * time.Millisecond
	const elevOnlineLimit = 3 * elevOnlineTick + 10*time.Millisecond
	const timeOutAcknowledge = 500 * time.Millisecond
	const doorOpentime = 500 * time.Millisecond
	var localIP string
	var externalOrders [N_FLOORS][2] ElevOrder
	var knownElevators = make(map[string]*Elevator) // IP address is key
	var activeElevators = make(map[string]bool) // IP address is key


	// Initializing hardware
	printDebug("Starting main loop")
	buttonChannel := make(chan hardware.ButtonEvent)
	lightChannel := make(chan hardware.LightEvent)
	motorChannel := make(chan int)
	floorChannel := make(chan int)
	if err := hardware.Init(buttonChannel, lightChannel, floorChannel, motorChannel, delayInPolling); err != nil {
		printDebug("Hardware Initializing failed")
		log.Fatal(err)
	} else {
		printDebug("Hardware Initializing successful")
	}

	// Initializing network
	receiveOrderChannel := make(chan OrderStruct, 5)
	sendOrderChannel := make(chan OrderStruct)
	receiveRecoveryChannel := make(chan BackupStruct, 5)
	sendRecoveryChannel := make(chan BackupStruct)

	localIP, err := networkInit(attemptToConnectLimit, receiveOrderChannel, sendOrderChannel, receiveRecoveryChannel, sendRecoveryChannel)
	if err != nil {
		printDebug("network Initializing failed")
		log.Fatal(err)
	} else {
		printDebug("Network Initializing successful")
	}

	// Initializing state
	printDebug("Requesting previous state")
	sendRecoveryChannel <- BackupStruct{
		RequesterIP: localIP
		Event: EventRequestState
	}
	knownElevators[localIP] = makeElevatorStruct(StateStruct{LocalIP: localIP, PrevFloor: <- floorChannel})
	setActiveElevators(knownElevators, activeElevators, localIP, elevOnlineLimit)
	printDebug("Finished initializing state, starting from floor: " knownElevators[localIP].State.LastFloor)


	// Initializing timers
	checkIfOnlineTicker := time.NewTicker(elevOnlineLimit)
	defer checkIfOnlineTicker.Stop()
	confirmOnlineTicker := time.NewTicker(elevOnlineTick)
	defer confirmOnlineTicker.Stop()
	doorTimer := time.NewTimer(time.Second)
	doorTimer.Stop()
	defer doorTimer.Stop()
	timeoutChannel := make(chan fullOrderStruct)
	printDebug("Ticker and timer initializing successful")

	// Main loop
	printDebug("Starting main loop")
	printDebug("\n\n\n")

	for{

		// Events happen in this select case
		select{

		//Hardware events
		case buttonEvent := <- buttonChannel:
			printDebug("Received a " + BUttonType[buttonEvent.type] + " from floor " + button.Floor + ". " + activeElevators.len() + " active elevators.")
			// A button is pressed
			switch button.Type{
				// External order
				case: BUTTON_CALL_UP, BUTTON_CALL_DOWN:
					if _, ok := activeElevators[localIP]; !ok {
						printDebug("Cannot accept external order while offline.")
					} else {
						// Do something with the order.
						if assignedIP, err := CostFunction.AssignNewOrder(knownElevators, activeElevators, button.Floor, button.Type); err != nil {
							log.Fatal(err)
						} else {
							sendOrderChannel <- OrderStruct{SendTo: assignedIP,
															SentFrom: localIP,
															Event: EventNewOrder,
															Floor: button.Floor,
															ButtonType: button.Type,
															}
						}
					}
				// Internal order
				case: BUTTON_COMMAND:
					if !knownElevators[localIP].State.IsMoving && knownElevators[localIP].State.LastFloor == button.Floor {
						// We are at a standstill at this floor
						var lightEvent := hardware.LightEvent{Type: DOOR_INDICATOR, Value: true}
						printDebug("Opening door")
						doorTimer.Reset(doorOpentime)
						knownElevators[localIP].State.OpenDoor = true
						var backupState := MakeBackupState(knownElevators[localIP], externalOrders)
					} else {
						printDebug("Internal order added to queue")
						knownElevators[localIP].SetInternalOrder(button.Floor)
						var backupState := MakeBackupState(knownElevators[localIP], externalOrders)
						var lightEvent := hardware.LightEvent{Type: button.Type, Floor: button.Floor, Value: true}
						if knownElevators[localIP].IsIdle() && !knownElevators[localIP].State.OpenDoor {
							doorTimer.Reset(0*time.Millisecond)
						}
					}
					lightChannel <- lightEvent
					sendRecoveryChannel <- backupState
				// Stop button pressed
				case BUTTON_STOP:
					motorChannel <- STOP
					lightChannel <- hardware.LightEvent{Type: BUTTON_STOP, Value: true}
					printDebug('\n\n\n')
					printDebug('Elevator was killed\n\n\n')
					time.Sleep(300*time.Millisecond)
					os.Exit(1)
				default:
					printDebug('Received button event from the hardware module')
			}
		case floorEvent := <-floorChannel:
			// Reached a floor
			printDebug("Reached floor: " + floorEvent.Floor)
			knownElevators[localIP].LastFloor = floor
			if knownElevators[localIP].ShouldStop(){
				// We are stopping at this floor
				motorChannel <- STOP
				knownElevators[localIP].SetMoving(false)
				printDebug("Opening doors")
				doorTimer.Reset(doorOpentime)
				lightChannel <- hardware.LightEvent{Type: DOOR_INDICATOR, Value: true}
				knownElevators[localIP].InternalOrders[floorEvent.Floor] = false
				lightChannel <- hardware.LightEvent{Floor: floor, Type: BUTTON_COMMAND, Value: false}
				if floorEvent.CurrentDirection == DIR_DOWN { 
					knownElevators[localIP].externalOrders[floorEvent.floor][0] = 0
					lightChannel <- hardware.LightEvent{Floor: floor, Type: BUTTON_CALL_DOWN, Value: false}
				} else if floorEvent.CurrentDirection == DIR_UP {
					knownElevators[localIP].externalOrders[floorEvent.floor][1] = 0
					lightChannel <- hardware.LightEvent{Floor: floor, Type: BUTTON_CALL_UP, Value: false}
				}
			}

		// Orders

		case order := receiveOrderChannel:
			printDebug("Received an " + EventType[order.Event] + " from " + order.SentFrom)

			switch order.Event {
			case EventNewOrder:
				printDebug("Order " + ButtonType[order.ButtonType] + " on floor " + strconv.Itoa(order.Floor))
				knownElevators[localIP].State.ExternalOrders[]


			case EventConfirmOrder:

			case EventAcknowledgeConfirmedOrder:

			case EventOrderDone:

			case EventAcknowledgeOrderDone:

			}







		// Timers
		case <- confirmOnlineTicker.C:
			sendRecoveryChannel <- MakeBackupMessage(knownElevators[localIP])

		case <- checkIfOnlineTicker.C:
			setActiveElevators(knownElevators, activeElevators, localIP, timeoutLimit)

		case <- doorTimer.C:
			printDebug("EventDoorTimeout")
			printDebug("Closing door ")
			knownElevators[localIP].State.OpenDoor = false
			lightChannel <- hardware.LightEvent{LightType: DOOR_LAMP, Value: false}

			// Check if we should start to move
			if knownElevators[localIP].State.HaveOrders(){
				knownElevators[localIP].SetDirection(knownElevators[localIP].GetNextDirection())
				knownElevators[localIP].SetMoving(knownElevators[localIP].State.Direction != DIR_STOP)
				printDebug("Have orders to execute")
				printDebug("Going " + MotorDirections[knownElevators[localIP].State.Direction + 1])
				lightChannel <- hardware.LightEvent(Floor: knownElevators[localIP].State.LastFloor, LightType: BUTTON_COMMAND, Value: false)
				motorChannel <- knownElevators[localIP].State.Direction
			} else {
				printDebug("Nothing to do")
				knownElevators[localIP].SetMoving(false)
				knownElevators[localIP].SetDirection(DIR_STOP)
			}
			sendRecoveryChannel <- MakeBackupMessage(knownElevators[localIP])
		}
	}
}

func networkInit(attemptToConnectLimit int, receiveOrderChannel, sendOrderChannel chan OrderStruct, receiveRecoveryChannel, sendRecoveryChannel chan StateStruct, timeoutLimit time.Duration){
	for i := 0; i <= attemptToConnectLimit: i++ {
		localIP, err := network.Init(receiveOrderChannel, sendOrderChannel, receiveRecoveryChannel, sendRecoveryChannel)
		if err != nil {
			if i == 0 {
				printDebug("Failed network Initializing, trying " + (attemptToConnectLimit - i).string() +" more times.")
			} else if i == attemptToConnectLimit {
				return "", err
			}
			time.Sleep(2*time.Second)
		} else {
			return localIP, nil
		}
	}
	return "", nil
}

func setActiveElevators(knownElevators map[string]*ElevatorStruct, activeElevators map[string]bool, localIP string){
	for key := range knownElevators{
		if time.Since(knownElevators[key].Time) =< timeoutLimit {
			if activeElevators[key] != true {
				activeElevators[key] = true
				printDebug("Added elevator " + knownElevators[key].State.LocalIP)
			}
		} else {
			if activeElevators[key] == true {
				printDebug("Removing elevator " + knownElevators[key].State.LocalIP + " from active elevators")
				delete(activeElevators, key)
			}
		}
	}
}



/*
	Helper function for debuggin
*/
func printDebug(message string){
	if debug{
		log.Println("Main:\t message")
	}
}