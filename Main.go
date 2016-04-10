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
	"fmt"
	"time"
	"udp"
	"hardware"
	"time"
	"network"
	. "./src/typedef"
)

func main() {
	
	// Order id can be generated from IP address + some counting variable

	const delayInPolling = 50 * time.Milliseconds
	const attemptToConnectLimit = 5
	const elevatorOnlineTick = 100 * time.Millisecond
	const elevatorOnlineLimit = 3 * elevOnlineTick + 10*time.Millisecond
	const accTimeoutLimit = 500 * time.Millisecond
	const doorOpentime = 500 * time.Millisecond
	const waitForBackupLimit = 2000 * time.Millisecond
	var localIP string
	var localState = StateStruct{}
	var knownElevators = make(map[string]*Elevator) // IP address is key
	var waitingBackups = make(map[string]*Elevator)
	var activeElevators = make(map[string]bool) // IP address is key
	var backupAccknowledgementList = make(map[string] bool)
	var waitingOrders = make(map[int] OrderStruct) // order ID is key
	var dispatchedOrders make(map[int] OrderStruct)
	var requestedBackup := false
	var stateChanged := false
	
	var waitingOrders = make(map[int]*OrderStruct)


	// Initializing hardware
	printDebug("Starting main loop")
	hardwareChannel := make(chan HardwareEvent)
	s
	if err := hardware.Init(hardwareChannel, delayInPolling); err != nil {
		printDebug("Hardware Initializing failed")
		log.Fatal(err)
	} else {
		printDebug("Hardware Initializing successful")
	}

	// Initializing network
	sendBackupChannel := make(chan<- ExtBackupEvent, 5)
	receiveBackupChannel := make(<-chan ExtBackupEvent)
	sendOrderChannel := make(chan<- ExtOrderEvent, 5)
	receiveOrderChannel := make(<- chan ExtOrderEvent)

	localIP, err := networkInit(attemptToConnectLimit, receiveOrderChannel, sendOrderChannel, receiveRecoveryChannel, sendRecoveryChannel)
	if err != nil {
		printDebug("network Initializing failed")
		log.Fatal(err)
	} else {
		printDebug("Network Initializing successful")
	}

	// Initializing state
	printDebug("Requesting previous state")
	sendRecoveryChannel <- ExtBackupStruct{
		RequesterIP: localIP
		Event: EventRequestStateFromElevator
	}
	// Blocking here until something is put on channel
	localState = StateStruct{LocalIP: localIP, PrevFloor: <- floorChannel}
	knownElevators[localIP] = makeElevatorStruct(localState)
	setActiveElevators(knownElevators, activeElevators, localIP, elevOnlineLimit)
	printDebug("Finished initializing state, starting from floor: " knownElevators[localIP].State.LastFloor)


	// Initializing timers
	checkIfOnlineTicker := time.NewTicker(elevatorOnlineLimit)
	defer checkIfOnlineTicker.Stop()
	confirmOnlineTicker := time.NewTicker(elevatorOnlineTick)
	defer confirmOnlineTicker.Stop()
	doorTimer := time.NewTimer(time.Second)
	doorTimer.Stop()
	defer doorTimer.Stop()
	printDebug("Ticker and timer initializing successful")

	// Main loop
	printDebug("Starting main loop")
	printDebug("\n\n\n")
	for{

		// Events happen in this select case
		select{

		//Hardware events
		case hwEvent := <- hardwareChannel:
			printDebug("Received a " + EventType[hwEvent.Event] + " from floor " + hwEvent.Floor + ". " + activeElevators.len() + " active elevators.")
			
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
						assignedIP, err := CostFunction.AssignNewOrder(knownElevators, activeElevators, button.Floor, button.Type)
						order := OrderStruct{OrderID: locOrderID++,
											SentFrom: localIP,
											Floor: button.Floor,
											Type: button.Type,
											}
						if err != nil {
							log.Fatal(err)
						} else {
							if assignedIP == localIP{
								addOrderToThisElevator(order)
							} else {
								dispatchedOrders[order.OrderID] = order
								dispatchedOrders[order.OrderID].DispatchedTime = time.Now()
								sendOrderChannel <- ExtOrderStruct{SendTo: assignedIP,
																	Order: order,
																	SentFrom: localIP,
																	Event: EventNewOrder
																	}
							}
						}
					}
				// Internal order
				case  BUTTON_COMMAND:
					if !knownElevators[localIP].State.IsMoving && knownElevators[localIP].State.LastFloor == button.Floor {
						// We are at a standstill at this floor
						var lightEvent := hardware.LightEvent{Type: DOOR_INDICATOR, Value: true}
						printDebug("Opening door")
						doorTimer.Reset(doorOpentime)
						knownElevators[localIP].State.OpenDoor = true
						var backupState := MakeBackupState(knownElevators[localIP], externalOrders)
					} else {
						printDebug("Internal order added to queue")
						localState.SetInternalOrder(button.Floor)
						knownElevators[localIP].State = localState
						stateChanged = true
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
				
				// A floor is reached	
		case hwEvent.EventFloorReached:
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
		}

		// Order events
		case extOrder := <-receiveOrderChannel:
			printDebug("Received an " + EventType[order.Event] + " from " + order.SentFrom)

			switch extOrder.Event {
			// Order receiver side
			case EventNewOrder:
				order := extOrder.Order
				printDebug("Order " + ButtonType[order.ButtonType] + " on floor " + strconv.Itoa(order.Floor))
				waitingOrders[order.OrderID] = order
				waitingOrders[order.OrderID].ReceivedTime = time.Now()
				// TODO -> Do something here to make a timer run to check wether the acc is being confirmed.
				// Accknowledge order
				sendOrderChannel <- ExtOrderStruct{SendTo: extOrder.sentFrom,
													SentFrom: localIP,
												    Event: EventAccOrderFromElevator}
			
			case EventConfirmAccFromElevator:
				if !m[extOrder.OrderID]{
					order = waitingOrders[extOrder.OrderID]
					addOrderToThisElevator(order)
					stateChanged = true
					delete(waitingOrders, extOrder.OrderID)
				}
				
			// Order dispatcher side
			case EventAccOrderFromElevator:
				delete(dispatchedOrders, extOrder.OrderID)
				sendOrderChannel <- ExtOrderStruct{SendTo: extOrder.SentFrom,
													 SentFrom: localIP,
													 Event: EventConfirmAccFromElevator}
			
			default:
				printDebug("Received unknown event in extOrder")	
				
			}
			
		// Backup
		case extBackup := <- receiveBackupChannel:
			printDebug("Received and " + EventType[extBackup.Event] + " from " + extBackup.SenderIP)
			
			
			switch extBackup.Event{
				// Backup receiver side
				case EventSendBackupToAll, EventStillOnline:
					// Received backup from someone..
					if extBackup.Event == EventSendBackupToAll{
						// Received updated state
						waitingBackups[extBackup.SentFrom] = extBackup.CurrentState
						waitingBackups[extBackup.SentFrom].BackupTime = time.Now()
						sendBackupChannel <- ExtBackupStruct{SentFrom: localIP, Event: EventAccBackup}
					}
					// Update last time backup was received
					knownElevators[extBackup.SentFrom].BackupTime = time.Now()
					
				case EventBackupAtAllConfirmed:
					backup := waitingBackups[extBackup.SentFrom]
					knownElevators[extBackup.SentFrom] = backup
					knownElevators[extBackup.SentFrom].BackupTime = time.Now()
					delete(waitingBackups, extBackup.SentFrom)
					
				// Backup sender side
				case extBackup.EventAccBackup:
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
					if knownElevators[extBackup.SentFrom]{
						sendBackupChannel <- ExtBackupStruct{Event: EventAnsweringBackupRequest,
																SendTo: extBackup.SentFrom,
																SentFrom: localIP,
																State knownElevators[extBackup.SentFrom].Backup
																}
					}
				case EventAnsweringBackupRequest:
					// We have received an answer to our backuprequest
					if hasRequstedBackup{
						localState = extBackup.Stae.CurrentState
						hasRequestedBackup  = false
					}			
			}
					
			
		// Timers
		case <- confirmOnlineTicker.C:
			sendRecoveryChannel <- ExtBackupStruct{Event: EventStillOnline,
													SentFrom: localIP,
													}

		case <- checkIfOnlineTicker.C:
			setActiveElevators(knownElevators, activeElevators, localIP, timeoutLimit)

		case <- checkBackupAccTimer.C:
			// Removed waiting backups that have timed out.
			for backupFromIP, backupData := range(waitingBackups){
				if time.Since(backupData.BackupTime) > backupLimit {
					delete(waitingBackups, backupFromIP)
				}
			}
		case <- checkOrderAccTimer.C:
			// Remove waiting orders that have timed out.
			for orderID, orderData := range(waitingOrders){
				if time.Since(orderData.ReceivedTime) > orderTimeoutLimit{
					delete(waitingOrders, orderID)
				}
			}
		case reassignOrderTimer.C:
			// Reassign order that has not been confirmed.
			for orderID, order := range(dispatchedOrders){
				if time.Since(order.DispatchedTime) > dispatchOrderTimeout{
					reassignOrder(order)
				}
			}
			

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
			if i == 0 {
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

func seedOrderID(localIP string) int {
	a := strings.Split(localIP, ".")
	b := a[len(a)-1]
	orderID, _ := strconv.Atoi(b)
	return orderID
}

func addOrderToThisElevator(order OrderStruct){
	switch order.Type{
		case BUTTON_DOWN:
			localState.ExternalOrders[order.Floor][0] = true
		case BUTTON_UP:
			localState.ExternalOrders[order.Floor][1] = true
	
	}
	knownElevator[localIP].BackupStruct.CurrentState = localState
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
		log.Println("Main:\t message")
	}
}