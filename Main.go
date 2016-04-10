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


	// Blocking here until something is put on channel
	printDebug("Blocking")
	firsFloorEvent := <- hardwareChannel
	localState = StateStruct{LocalIP: localIP, PrevFloor: firsFloorEvent.Floor,}
	knownElevators[localIP] = &Elevator{State: localState,}
	setActiveElevators(knownElevators, activeElevators, localIP, elevatorOnlineTimeout)
	printDebug("Finished initializing state, starting from floor: " + strconv.Itoa(knownElevators[localIP].State.PrevFloor))



	// Initializing network
	sendBackupChannel := make(chan ExtBackupStruct, 5)
	receiveBackupChannel := make(chan ExtBackupStruct)
	sendOrderChannel := make(chan ExtOrderStruct, 5)
	receiveOrderChannel := make(chan ExtOrderStruct)

	localIP, err := networkInit(connectionLimit, receiveOrderChannel, sendOrderChannel, receiveBackupChannel, sendBackupChannel, 2*time.Second)
	locOrderID = seedOrderID(localIP)
	if err != nil {
		printDebug("network Initializing failed")
		log.Fatal(err)
	} else {
		printDebug("Network Initializing successful")
	}


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

	checkBackupAccTimer := time.NewTicker(checkBackupAccTick)
	defer checkBackupAccTimer.Stop()

	checkOrderAccTimer := time.NewTicker(checkOrderAccTick)
	defer checkOrderAccTimer.Stop()

	reassignOrderTimer := time.NewTicker(dispatchOrderTick)
	defer reassignOrderTimer.Stop()

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
								addOrderToThisElevator(order, localState, knownElevators, localIP)
								if localState.CurrentDirection == DIR_STOP{
									localState.SetDirection(knownElevators[localIP].GetNextDirection())
									localState.Moving = true
									knownElevators[localIP].State = localState
									hardware.SetMotorDirection(localState.CurrentDirection)
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
					lightEvent := hardware.LightEvent{}
					if !knownElevators[localIP].State.Moving && knownElevators[localIP].State.PrevFloor == hwEvent.Floor {
						// We are at a standstill at this floor
						lightEvent = hardware.LightEvent{LightType: DOOR_LAMP, Value: true}
						printDebug("Opening door")
						doorTimer.Reset(doorOpentime)
						knownElevators[localIP].State.OpenDoor = true
					} else {
						printDebug("Internal order added to queue: " + strconv.Itoa(hwEvent.Floor))
						localState.InternalOrders[hwEvent.Floor] = true
						knownElevators[localIP].State = localState
						if localState.CurrentDirection == DIR_STOP{
							localState.SetDirection(knownElevators[localIP].GetNextDirection())
							localState.Moving = true
							knownElevators[localIP].State = localState
							hardware.SetMotorDirection(localState.CurrentDirection)
						}
						stateChanged = true
						lightEvent = hardware.LightEvent{LightType: hwEvent.ButtonType, Floor: hwEvent.Floor, Value: true}
					}
					hardware.SetLights(lightEvent)
				// Stop button pressed
				case BUTTON_STOP:
					hardware.SetMotorDirection(DIR_STOP)
					localState.SetMoving(false)
					localState.SetDirection(DIR_STOP)
					hardware.SetLights(hardware.LightEvent{LightType: BUTTON_STOP, Value: true,})
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
			printDebug("Reached floor: " + strconv.Itoa(hwEvent.Floor))
			knownElevators[localIP].State.PrevFloor = hwEvent.Floor
			if knownElevators[localIP].ShouldStop(){
				// We are stopping at this floor
				hardware.SetMotorDirection(DIR_STOP)
				localState.SetMoving(false)
				localState.SetDirection(DIR_STOP)
				knownElevators[localIP].State = localState
				printDebug("Opening doors")
				doorTimer.Reset(doorOpentime)
				hardware.SetLights(hardware.LightEvent{LightType: DOOR_LAMP, Value: true})
				knownElevators[localIP].State.InternalOrders[hwEvent.Floor] = false
				hardware.SetLights(hardware.LightEvent{Floor: hwEvent.Floor, LightType: BUTTON_COMMAND, Value: false})
				if hwEvent.CurrentDirection == DIR_DOWN { 
					knownElevators[localIP].State.ExternalOrders[0][hwEvent.Floor] = false
					hardware.SetLights(hardware.LightEvent{Floor: hwEvent.Floor, LightType: BUTTON_CALL_DOWN, Value: false})
				} else if hwEvent.CurrentDirection == DIR_UP {
					knownElevators[localIP].State.ExternalOrders[1][hwEvent.Floor] = false
					hardware.SetLights(hardware.LightEvent{Floor: hwEvent.Floor, LightType: BUTTON_CALL_UP, Value: false})
				}
			}
		}

		// Order events
		case extOrder := <-receiveOrderChannel:
			printDebug("Received an " + EventType[extOrder.Event] + " from " + extOrder.SentFrom)

			switch extOrder.Event {
			// Order receiver side
			case EventSendOrderToElevator:
				order := extOrder.Order
				printDebug("Order " + HardwareEventType[order.Type] + " on floor " + strconv.Itoa(order.Floor))
				waitingOrders[order.OrderID] = &order
				waitingOrders[order.OrderID].ReceivedTime = time.Now()
				// TODO -> Do something here to make a timer run to check wether the acc is being confirmed.
				// Accknowledge order
				sendOrderChannel <- ExtOrderStruct{SendTo: extOrder.SentFrom,
													SentFrom: localIP,
												    Event: EventAccOrderFromElevator}
			
			case EventConfirmAccFromElevator:
				if _, ok := waitingOrders[extOrder.Order.OrderID]; ok {
					order := waitingOrders[extOrder.Order.OrderID]
					addOrderToThisElevator(*order, localState, knownElevators, localIP)
					if localState.CurrentDirection == DIR_STOP{
						localState.SetDirection(knownElevators[localIP].GetNextDirection())
						localState.Moving = true
						knownElevators[localIP].State = localState
						hardware.SetMotorDirection(localState.CurrentDirection)
					}
					stateChanged = true
					delete(waitingOrders, extOrder.Order.OrderID)
				}
				
			// Order dispatcher side
			case EventAccOrderFromElevator:
				delete(dispatchedOrders, extOrder.Order.OrderID)
				sendOrderChannel <- ExtOrderStruct{SendTo: extOrder.SentFrom,
													 SentFrom: localIP,
													 Event: EventConfirmAccFromElevator}
			
			default:
				printDebug("Received unknown event in extOrder")	
				
			}
			
		// Backup
		case extBackup := <- receiveBackupChannel:
			printDebug("Received an " + EventType[extBackup.Event] + " from " + extBackup.SentFrom)
			
			
			switch extBackup.Event{
				// Backup receiver side
				case EventSendBackupToAll, EventStillOnline:
					// Received backup from someone..
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

		case <- checkBackupAccTimer.C:
			// Removed waiting backups that have timed out.
			for backupFromIP, backupData := range(waitingBackups){
				if time.Since(backupData.BackupTime) > backupTimeout {
					printDebug("Waiting backup timed out, removing backup from: " + backupFromIP)
					delete(waitingBackups, backupFromIP)
				}
			}
		case <- checkOrderAccTimer.C:
			// Remove waiting orders that have timed out.
			for orderID, orderData := range(waitingOrders){
				if time.Since(orderData.ReceivedTime) > orderTimeout{
					printDebug("Waiting order timed out, removing id: " + strconv.Itoa(orderID))
					delete(waitingOrders, orderID)
				}
			}
		case <-reassignOrderTimer.C:
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
							addOrderToThisElevator(*order, localState, knownElevators, localIP)
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
		if time.Since(knownElevators[key].Time) >= timeoutLimit {
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

func seedOrderID(localIP string) int {
	a := strings.Split(localIP, ".")
	b := a[len(a)-1]
	orderID, _ := strconv.Atoi(b)
	return orderID
}

func addOrderToThisElevator(order OrderStruct, localState StateStruct, knownElevator map[string]*Elevator, localIP string){
	switch order.Type{
		case BUTTON_CALL_DOWN:
			localState.ExternalOrders[order.Floor][0] = true
		case BUTTON_CALL_UP:
			localState.ExternalOrders[order.Floor][1] = true
	
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