package main

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
	"time"
	"log"
	"strconv"
	"strings"
	"./src/hardware"
	"./src/network"
	. "./src/typedef"
	"os"
)

const debug bool = true

func main() {
	
	// Order id can be generated from IP address + some counting variable


	const pollingDelay = 50 * time.Millisecond
	const connectionLimit = 5

	const checkElevatorsTick = 100 * time.Millisecond
	const checkBackupAccTick = 500 * time.Millisecond
	const checkOrderAccTick = 500 * time.Millisecond
	const dispatchOrderTick = 500 * time.Millisecond
	const thisElevatorOnlineTick = 100 * time.Millisecond
	
	const elevatorOnlineTimeout = 3 * checkElevatorsTick
	const backupTimeout = 3 * checkBackupAccTick
	const orderTimeout = 3 * checkOrderAccTick
	const dispatchOrderTimeout = 3 * dispatchOrderTick
	
	const initialBackupRequestTimeout = 3 * time.Second
	const doorOpentime = 1000 * time.Millisecond
	

	var localIP string
	var localState = StateStruct{}
	var stateChanged bool 
	var locOrderID int

	var knownElevators = make(map[string]*Elevator) // IP address is key
	var waitingBackups = make(map[string]*BackupStruct)
	var activeElevators = make(map[string]bool) // IP address is key
	var backupAccknowledgementList = make(map[string] bool)
	var waitingOrders = make(map[int] *OrderStruct) // order ID is key
	var dispatchedOrders = make(map[int] *OrderStruct)


	// Initializing hardware
	printDebug("Initializing hardware")
	hardwareChannel := make(chan HardwareEvent, 10)
	if err := hardware.Init(hardwareChannel, pollingDelay); err != nil {
		printDebug("Hardware Initializing failed")
		log.Fatal(err)
	} else {
		printDebug("Hardware Initializing successful")
	}


	// Blocking here until hwEvent is received, containing current floor.
	printDebug("Blocking")
	getfloorstate:
	for{
		select {
		case hwEvent:= <- hardwareChannel:
			localState.PrevFloor = hwEvent.Floor
			break getfloorstate
		}
	}

	// Initializing network
	sendBackupChannel := make(chan ExtBackupStruct, 5)
	receiveBackupChannel := make(chan ExtBackupStruct)
	sendOrderChannel := make(chan ExtOrderStruct, 5)
	receiveOrderChannel := make(chan ExtOrderStruct)

	localIP, err := networkInit(connectionLimit, receiveOrderChannel, sendOrderChannel, receiveBackupChannel, sendBackupChannel, 2*time.Second)
	locOrderID = seedOrderID(localIP)*1000
	if err != nil {
		printDebug("network Initializing failed")
		log.Fatal(err)
	} else {
		printDebug("Network Initializing successful")
	}

	// Finish localState and make this elevator 'known' to itself.
	localState.LocalIP = localIP 
	knownElevators[localIP] = &Elevator{State: localState,}
	setActiveElevators(knownElevators, activeElevators, localIP, elevatorOnlineTimeout)
	printDebug("Finished initializing state, starting from floor: " + strconv.Itoa(knownElevators[localIP].State.PrevFloor))


	// Initializing state
	printDebug("Requesting previous state")
	sendBackupChannel <- ExtBackupStruct{
		RequesterIP: localIP,
		Event: EventRequestStateFromElevator,
	}
	hasRequestedBackup := true

	L: // This will make the break statement in the loop break both the select and the for
	for{
		select{
		case extBackup := <- receiveBackupChannel:
			if extBackup.Event == EventAnsweringBackupRequest{
					// We have received an answer to our backuprequest
				if hasRequestedBackup{

					// Set the current state to the state received here.
					localState.InternalOrders = extBackup.BackupData.CurrentState.InternalOrders
					hasRequestedBackup  = false
					printDebug("Received backup from " + extBackup.SentFrom)
				}
			}
		case <- time.After(initialBackupRequestTimeout):
			printDebug("Backup request timed out")
			hasRequestedBackup = false
			break L
		}
	}


	// Initializing timers
	checkIfOnlineTicker := time.NewTicker(checkElevatorsTick)
	defer checkIfOnlineTicker.Stop()

	confirmOnlineTicker := time.NewTicker(thisElevatorOnlineTick)
	defer confirmOnlineTicker.Stop()

	checkBackupAccTicker := time.NewTicker(checkBackupAccTick)
	defer checkBackupAccTicker.Stop()

	checkOrderAccTicker := time.NewTicker(checkOrderAccTick)
	defer checkOrderAccTicker.Stop()

	reassignOrderTicker := time.NewTicker(dispatchOrderTick)
	defer reassignOrderTicker.Stop()

	doorTimer := time.NewTimer(time.Second)
	doorTimer.Stop()
	defer doorTimer.Stop()

	printDebug("Ticker and timer initialized successful")

	// Main loop
	printDebug("Starting main loop")
	printDebug("\n\n\n")
	for{

		// Events happen in this select case
		select{

		//Hardware events
		case hwEvent := <- hardwareChannel:
			printDebug("Received a " + EventType[hwEvent.Event] + " from floor " + strconv.Itoa(hwEvent.Floor) + ". " + strconv.Itoa(len(activeElevators)) + " active elevators.")
			
			switch hwEvent.Event{
				
				// A button is pressed
				case EventButtonPressed:
					switch hwEvent.ButtonType{

					// External order
					case BUTTON_CALL_UP, BUTTON_CALL_DOWN:
						if _, ok := activeElevators[localIP]; !ok {
							printDebug("Cannot accept external order while offline.")
						} else {
							// Do something with the order.
							assignedIP := localIP //CostFunction.AssignNewOrder(knownElevators, activeElevators, button.Floor, button.Type)
							var err error
							order := OrderStruct{OrderID: locOrderID,
												Floor: hwEvent.Floor,
												Type: hwEvent.ButtonType,
												}
							locOrderID = locOrderID + 1
							if err != nil {
								log.Fatal(err)
							} else {
								if assignedIP == localIP{
									addOrderToThisElevator(*order, &localState, knownElevators, localIP)
									if localState.CurrentDirection == DIR_STOP{
										localState.SetDirection(knownElevators[localIP].GetNextDirection())
										localState.SetMoving(true)
										knownElevators[localIP].State = localState
										hardware.SetMotorDirection(localState.CurrentDirection)
										hardware.SetLights(order.Type, order.Floor, true)
									}
									stateChanged = true
								} else {
									dispatchedOrders[order.OrderID] = &order
									dispatchedOrders[order.OrderID].DispatchedTime = time.Now()
									sendOrderChannel <- ExtOrderStruct{SendTo: assignedIP,
																		Order: order,
																		SentFrom: localIP,
																		Event: EventSendOrderToElevator,
																		}
								}
							}
						}

					// Internal order
					case  BUTTON_COMMAND:
						if !knownElevators[localIP].State.Moving && knownElevators[localIP].State.PrevFloor == hwEvent.Floor {
							// We are at a standstill at this floor
							printDebug("Opening door")
							doorTimer.Reset(doorOpentime)
							hardware.SetLights(DOOR_LAMP, 0, true)
							knownElevators[localIP].State.OpenDoor = true
						} else {
							printDebug("Internal order added to queue: " + strconv.Itoa(hwEvent.Floor))
							localState.InternalOrders[hwEvent.Floor] = true
							knownElevators[localIP].State = localState
							if localState.CurrentDirection == DIR_STOP && !localState.OpenDoor{
								localState.SetDirection(knownElevators[localIP].GetNextDirection())
								localState.Moving = true
								knownElevators[localIP].State = localState
								hardware.SetMotorDirection(localState.CurrentDirection)
							}
							stateChanged = true
							hardware.SetLights(hwEvent.ButtonType, hwEvent.Floor, true)
						}

					// Stop button pressed, can perhaps remove this?
					case BUTTON_STOP:
						hardware.SetMotorDirection(DIR_STOP)
						printDebug("\n\n\n")
						printDebug("Elevator was killed\n\n\n")
						time.Sleep(300*time.Millisecond)
						os.Exit(1)
					default:
						printDebug("Received button event from the hardware module")
					}
				
				// A floor is reached	
		case EventFloorReached:
			// Reached a floor
			printDebug(localIP + " Reached floor: " + strconv.Itoa(hwEvent.Floor))
			localState.PrevFloor = hwEvent.Floor
			knownElevators[localIP].State = localState
			if knownElevators[localIP].ShouldStop(){
				
				// We are stopping at this floor
				hardware.SetMotorDirection(DIR_STOP)
				localState.SetMoving(false)
				localState.SetDirection(DIR_STOP)
	
				printDebug("Opening doors")
				doorTimer.Reset(doorOpentime)
				hardware.SetLights(DOOR_LAMP, 0, true)
				
				localState.InternalOrders[hwEvent.Floor] = false
				hardware.SetLights(BUTTON_COMMAND, hwEvent.Floor, false)
				if hwEvent.CurrentDirection == DIR_DOWN { 
					localState.ExternalOrders[0][hwEvent.Floor] = false
					hardware.SetLights(BUTTON_CALL_DOWN, hwEvent.Floor, false)
				} else if hwEvent.CurrentDirection == DIR_UP {
					localState.ExternalOrders[1][hwEvent.Floor] = false
					hardware.SetLights(BUTTON_CALL_UP, hwEvent.Floor, false)
				}
				knownElevators[localIP].State = localState
			}
		}

		// Order events
		case extOrder := <-receiveOrderChannel:
			printDebug("Received an " + EventType[extOrder.Event] + " from " + extOrder.SentFrom)

			switch extOrder.Event {
			// Received new order
			case EventSendOrderToElevator:
				order := extOrder.Order
				printDebug("Order " + HardwareEventType[order.Type] + " on floor " + strconv.Itoa(order.Floor))
				waitingOrders[order.OrderID] = &order
				waitingOrders[order.OrderID].ReceivedTime = time.Now()
				
				// Accknowledge order
				sendOrderChannel <- ExtOrderStruct{SendTo: extOrder.SentFrom,
													SentFrom: localIP,
													OrderID: order.OrderID,
												    Event: EventAccOrderFromElevator}
			
			// Received confirmation on acc, can execute order now.
			case EventConfirmAccFromElevator:
				if _, ok := waitingOrders[extOrder.OrderID]; ok {
					order := waitingOrders[extOrder.OrderID]
					addOrderToThisElevator(*order, &localState, knownElevators, localIP)
					if localState.CurrentDirection == DIR_STOP{
						localState.SetDirection(knownElevators[localIP].GetNextDirection())
						localState.Moving = true
						knownElevators[localIP].State = localState
						hardware.SetMotorDirection(localState.CurrentDirection)
					}
					stateChanged = true
					delete(waitingOrders, extOrder.OrderID)
				}
				
			// Received acc on dispatched order
			case EventAccOrderFromElevator:
				delete(dispatchedOrders, extOrder.OrderID)
				sendOrderChannel <- ExtOrderStruct{SendTo: extOrder.SentFrom,
													OrderID: extOrder.OrderID,
													 SentFrom: localIP,
													 Event: EventConfirmAccFromElevator}
			
			default:
				printDebug("Received unknown event in extOrder")	
				
			}
			
		// Backup
		case extBackup := <- receiveBackupChannel:
			printDebug("Received an " + EventType[extBackup.Event] + " from " + extBackup.SentFrom)
			
			switch extBackup.Event{

				// Received new backupData, or confirmation that still online
				case EventSendBackupToAll, EventStillOnline:
					if extBackup.Event == EventSendBackupToAll{
						// Received updated state
						waitingBackups[extBackup.SentFrom] = &extBackup.BackupData
						waitingBackups[extBackup.SentFrom].BackupTime = time.Now()
						sendBackupChannel <- ExtBackupStruct{SentFrom: localIP, Event: EventAccBackup}
					}
					// Update last time backup was received
					knownElevators[extBackup.SentFrom].Time = time.Now()
					
				case EventBackupAtAllConfirmed:
					backup := waitingBackups[extBackup.SentFrom]
					knownElevators[extBackup.SentFrom].State = backup.CurrentState
					knownElevators[extBackup.SentFrom].Time = time.Now()
					delete(waitingBackups, extBackup.SentFrom)
					
				// Backup sender side
				case EventAccBackup:
					// Someone accknowledged our backup
					backupAccknowledgementList[extBackup.SentFrom] = true
					if allOtherAccBackup(backupAccknowledgementList, activeElevators) {
						// Everyone has received backup
						stateChanged = false
						sendBackupChannel <- ExtBackupStruct{SentFrom: localIP, Event: EventBackupAtAllConfirmed}
					}
					
				// Backup Requests
				case EventRequestStateFromElevator:
				//  SendAnswer to backup request
					if _,ok := knownElevators[extBackup.SentFrom]; ok{
						sendBackupChannel <- ExtBackupStruct{Event: EventAnsweringBackupRequest,
																SendTo: extBackup.SentFrom,
																SentFrom: localIP,
																BackupData: BackupStruct{CurrentState: knownElevators[extBackup.SentFrom].State,},
																}
					}
							
			}
					
			
		// Timers
		case <- confirmOnlineTicker.C:
			if stateChanged{
				sendBackupChannel <- ExtBackupStruct{Event: EventSendBackupToAll,
													 SentFrom: localIP,
													 BackupData: BackupStruct{CurrentState: localState,},
													}
			} else {
				sendBackupChannel <- ExtBackupStruct{Event: EventStillOnline,
													SentFrom: localIP,
													}
			}
			

		case <- checkIfOnlineTicker.C:
			setActiveElevators(knownElevators, activeElevators, localIP, elevatorOnlineTimeout)

		case <- checkBackupAccTicker.C:
			// Removed waiting backups that have timed out.
			for backupFromIP, backupData := range(waitingBackups){
				if time.Since(backupData.BackupTime) > backupTimeout {
					printDebug("Waiting backup timed out, removing backup from: " + backupFromIP)
					delete(waitingBackups, backupFromIP)
				}
			}
		case <- checkOrderAccTicker.C:
			// Remove waiting orders that have timed out.
			for orderID, orderData := range(waitingOrders){
				if time.Since(orderData.ReceivedTime) > orderTimeout{
					printDebug("Waiting order timed out, removing id: " + strconv.Itoa(orderID))
					delete(waitingOrders, orderID)
				}
			}
		case <-reassignOrderTicker.C:
			// Reassign order that has not been confirmed.
			for orderID, order := range(dispatchedOrders){
				if time.Since(order.DispatchedTime) > dispatchOrderTimeout{
					printDebug("Dispatched order timed out, reassigning id: " + strconv.Itoa(orderID))
					assignedIP := localIP //CostFunction.AssignNewOrder(knownElevators, activeElevators, button.Floor, button.Type)
					var err error
					order := dispatchedOrders[orderID]
					order.OrderID = locOrderID
					locOrderID = locOrderID + 1
					delete(dispatchedOrders, orderID)
					if err != nil {
						log.Fatal(err)
					} else {
						if assignedIP == localIP{
							addOrderToThisElevator(*order, &localState, knownElevators, localIP)
							if localState.CurrentDirection == DIR_STOP{
								localState.SetDirection(knownElevators[localIP].GetNextDirection())
								localState.Moving = true
								knownElevators[localIP].State = localState
								hardware.SetMotorDirection(localState.CurrentDirection)
							}
							stateChanged = true
						} else {
							dispatchedOrders[order.OrderID] = order
							dispatchedOrders[order.OrderID].DispatchedTime = time.Now()
							sendOrderChannel <- ExtOrderStruct{SendTo: assignedIP,
																Order: *order,
																SentFrom: localIP,
																Event: EventSendOrderToElevator,
																}
						}
					}
				}
			}
			
		case <- doorTimer.C:
			printDebug("Closing door ")
			knownElevators[localIP].State.OpenDoor = false
			hardware.SetLights(hardware.LightEvent{LightType: DOOR_LAMP, Value: false})

			// Check if we should start to move
			if knownElevators[localIP].State.HaveOrders(){
				knownElevators[localIP].SetDirection(knownElevators[localIP].GetNextDirection())
				knownElevators[localIP].SetMoving(knownElevators[localIP].State.CurrentDirection != DIR_STOP)
				localState = knownElevators[localIP].State
				printDebug("Going " + MotorDirections[knownElevators[localIP].State.CurrentDirection + 1])
				hardware.SetLights(hardware.LightEvent{Floor: knownElevators[localIP].State.PrevFloor, LightType: BUTTON_COMMAND, Value: false})
				hardware.SetMotorDirection(knownElevators[localIP].State.CurrentDirection)
			} else {
				printDebug("Nothing to do")
				knownElevators[localIP].SetMoving(false)
				knownElevators[localIP].SetDirection(DIR_STOP)
			}
			sendBackupChannel <- MakeBackupMessage(knownElevators[localIP])
		}
	}
}

func networkInit(connectionLimit int, receiveOrderChannel chan ExtOrderStruct, sendOrderChannel chan ExtOrderStruct, receiveRecoveryChannel chan ExtBackupStruct, sendRecoveryChannel chan ExtBackupStruct, timeoutLimit time.Duration)(string, error){
	for i := 0; i <= connectionLimit; i++ {
		localIP, err := network.Init(receiveOrderChannel, receiveRecoveryChannel, sendOrderChannel, sendRecoveryChannel)
		if err != nil {
			if i == 0 {
				printDebug("Failed network Initializing, trying " + strconv.Itoa(connectionLimit - i) +" more times.")
			} else if i == connectionLimit {
				return "", err
			}
			time.Sleep(2*time.Second)
		} else {
			return localIP, nil
		}
	}
	return "", nil
}

func setActiveElevators(knownElevators map[string]*Elevator, activeElevators map[string]bool, localIP string, timeoutLimit time.Duration){
	for key := range knownElevators{
		if key == localIP{
			activeElevators[key] = true
		}else if time.Since(knownElevators[key].Time) >= timeoutLimit {
			if activeElevators[key] != true {
				activeElevators[key] = true
				printDebug("Added active elevator " + knownElevators[key].State.LocalIP)
			}
		} else {
			if activeElevators[key] == true {
				printDebug("Removing inactive elevator " + knownElevators[key].State.LocalIP)
				delete(activeElevators, key)
			}
		}
	}
}

func seedOrderID(localIP string) int {
	a := strings.Split(localIP, ".")
	b := a[len(a)-1]
	orderID, _ := strconv.Atoi(b)
	return orderID
}

func addOrderToThisElevator(order OrderStruct, localState *StateStruct, knownElevator map[string]*Elevator, localIP string){
	switch order.Type{
		case BUTTON_CALL_DOWN:
			localState.ExternalOrders[0][order.Floor] = true
		case BUTTON_CALL_UP:
			localState.ExternalOrders[1][order.Floor] = true
	
	}
	knownElevator[localIP].State = localState
}

func allOtherAccBackup(backupAcknowledgementList, activeElevators map[string]bool) bool {
	for elevatorIP, active := range activeElevators{
		if active{
			if !backupAcknowledgementList[elevatorIP]{
				return false
			}
		}
	}
	return true
}


/*
	Helper function for debuggin
*/
func printDebug(message string){
	if debug{
		log.Println("Main:\t" + message)
	}
}