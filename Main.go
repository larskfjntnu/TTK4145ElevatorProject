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
	"./src/costFunction"
	"strconv"
	"strings"
	"./src/hardwareSim"
	"./src/network"
	. "./src/typedef"
	"os"
)

const debug bool = true

func main() {
	defer func(){
		hardwareSim.SetMotorDirection(DIR_STOP)
	}()
	// Order id can be generated from IP address + some counting variable

	const pollingDelay = 50 * time.Millisecond
	const connectionLimit = 5

	const checkElevatorsTick = 100 * time.Millisecond
	const checkBackupAccTick = 500 * time.Millisecond
	const checkOrderAccTick = 500 * time.Millisecond
	const dispatchOrderTick = 500 * time.Millisecond
	const thisElevatorOnlineTick = 100 * time.Millisecond
	
	const elevatorOnlineTimeout = 30 * checkElevatorsTick
	const backupTimeout = 30 * checkBackupAccTick
	const orderTimeout = 30 * checkOrderAccTick
	const dispatchOrderTimeout = 30 * dispatchOrderTick
	
	const initialBackupRequestTimeout = 3000 * time.Millisecond
	const doorOpentime = 1000 * time.Millisecond
	

	var localIP string
	var localState = StateStruct{}
	var stateChanged bool 
	var locOrderID int

	var knownElevators = make(map[string]*Elevator) // IP address is key
	var waitingBackups = make(map[string]*BackupStruct)
	var haveBackups = make(map[string]bool)
	var activeElevators = make(map[string]bool) // IP address is key
	var backupAccknowledgementList = make(map[string] bool)
	var waitingOrders = make(map[int] *OrderStruct) // order ID is key
	var dispatchedOrders = make(map[int] *OrderStruct)


	// Initializing hardware
	hardwareChannel := make(chan HardwareEvent, 10)
	if err := hardwareSim.Init(hardwareChannel, pollingDelay); err != nil {
		printDebug("Hardware Initializing failed")
		log.Fatal(err)
	} else {
		printDebug("Hardware Initializing successful")
	}


	// Blocking here until hwEvent is received, containing current floor.
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
	sendBackupChannel <- ExtBackupStruct{
		SentFrom: localIP,
		Event: EventRequestStateFromElevator,
	}
	hasRequestedBackup := true
	printDebug("Waiting for backup")
	
	
	t := time.After(3000*time.Millisecond)
	L: // This will make the break statement in the loop break both the select and the for
	for{
		select{
		case extBackup := <- receiveBackupChannel:
			if extBackup.Event == EventAnsweringBackupRequest{
					// We have received an answer to our backuprequest
				printDebug("Received backup from " + extBackup.SentFrom)
				if hasRequestedBackup{

					// Set the current state to the state received here.
					knownElevators[localIP].State.InternalOrders = extBackup.BackupData.CurrentState.InternalOrders
					knownElevators[localIP].SetDirection(DIR_STOP)
					hasRequestedBackup  = false
					printDebug("Received backup from " + extBackup.SentFrom)
				}
				break L
			}
		case <- t:
			printDebug("Backup request timed out")
			hasRequestedBackup = false
			break L
		}

	}
	printDebug("Initializing done...........!")

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
	printDebug("\n\n\n")
	for{

		// Events happen in this select case
		select{

		//Hardware events
		case hwEvent := <- hardwareChannel:
			printDebug("Received a " + EventType[hwEvent.Event] + " from floor " + strconv.Itoa(hwEvent.Floor))
			
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
							assignedIP, err := costFunction.CalculateRespondingElevator(knownElevators, activeElevators, localIP, hwEvent.ButtonType, hwEvent.Floor)
							order := OrderStruct{OrderID: locOrderID,
												Floor: hwEvent.Floor,
												Type: hwEvent.ButtonType,
												}
							printDebug("Order: " + strconv.Itoa(locOrderID) +  " assigned to:  " + assignedIP)
							locOrderID = locOrderID + 1
							if err != nil {
								log.Fatal(err)
							} else {
								if assignedIP == localIP{
									if knownElevators[localIP].GetDirection() == DIR_STOP && order.Floor == knownElevators[localIP].GetFloor(){
										printDebug("Opening door")
										openDoor(doorTimer, doorOpentime, knownElevators, localIP)
									} else {
										editOrderOnThisElevator(order.Floor, order.Type, true, knownElevators, localIP)
										log.Println(knownElevators[localIP].MakeQueue())
										if knownElevators[localIP].GetDirection() == DIR_STOP{
											startMoving(knownElevators, localIP)
										}
									}
									stateChanged = true
								} else {
									dispatchOrder(order, assignedIP, localIP, dispatchedOrders, sendOrderChannel)
								}
							}
						}

					// Internal order
					case  BUTTON_COMMAND:
						if !knownElevators[localIP].IsMoving() && knownElevators[localIP].State.PrevFloor == hwEvent.Floor {
							// We are at a standstill at this floor
							printDebug("Opening door")
							openDoor(doorTimer, doorOpentime, knownElevators, localIP)
						} else {
							knownElevators[localIP].SetInternalOrder(hwEvent.Floor, true)
							hardwareSim.SetLights(hwEvent.ButtonType, hwEvent.Floor, true)
							if knownElevators[localIP].GetDirection() == DIR_STOP && !knownElevators[localIP].DoorOpen(){
								startMoving(knownElevators, localIP)
							}
							stateChanged = true
							log.Println(knownElevators[localIP].MakeQueue())
						}

					// Stop button pressed, can perhaps remove this?
					case BUTTON_STOP:
						hardwareSim.SetMotorDirection(DIR_STOP)
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
				knownElevators[localIP].SetFloor(hwEvent.Floor)
				if knownElevators[localIP].ShouldStop(){
					
					// We are stopping at this floTor
					stopMoving(knownElevators, localIP, hwEvent.Floor)
		
					printDebug("Opening doors")
					openDoor(doorTimer, doorOpentime, knownElevators, localIP)
					
					knownElevators[localIP].SetInternalOrder(hwEvent.Floor, false)
					hardwareSim.SetLights(BUTTON_COMMAND, hwEvent.Floor, false)
					if hwEvent.CurrentDirection == DIR_DOWN || (hwEvent.CurrentDirection == DIR_UP && !knownElevators[localIP].State.OrdersAbove()) { 
						editOrderOnThisElevator(hwEvent.Floor, BUTTON_CALL_DOWN, false, knownElevators, localIP)
					}
					if hwEvent.CurrentDirection == DIR_UP || (hwEvent.CurrentDirection == DIR_DOWN && !knownElevators[localIP].State.OrdersBelow()) {
						editOrderOnThisElevator(hwEvent.Floor, BUTTON_CALL_UP, false, knownElevators, localIP)
					}
					log.Println(knownElevators[localIP].MakeQueue())
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
					editOrderOnThisElevator(order.Floor, order.Type, true, knownElevators, localIP)
					log.Println(knownElevators[localIP].MakeQueue())
					if knownElevators[localIP].GetDirection() == DIR_STOP && !knownElevators[localIP].DoorOpen() {
						startMoving(knownElevators, localIP)
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
			
			if _, ok := knownElevators[extBackup.SentFrom]; !ok {
				// New elevator on network, add it and push our state again.
				
				newElev := Elevator{Time: time.Now(),}
				if extBackup.Event == EventSendBackupToAll{
					newElev.State = extBackup.BackupData.CurrentState
				} else {
					newElev.State = StateStruct{LocalIP: extBackup.SentFrom,}
				}
				knownElevators[extBackup.SentFrom] = &newElev
				activeElevators[extBackup.SentFrom] = true
				stateChanged = true;
				printDebug("Added elevator " + extBackup.SentFrom + " to knownElevators")
			}
			switch extBackup.Event{

			// Received new backupData, or confirmation that still online
			case EventSendBackupToAll, EventStillOnline:
				if extBackup.Event == EventSendBackupToAll {
					// Received updated state
					printDebug("Adding backup of:	" + extBackup.SentFrom)
					waitingBackups[extBackup.SentFrom] = &extBackup.BackupData
					waitingBackups[extBackup.SentFrom].BackupTime = time.Now()
					sendBackupChannel <- ExtBackupStruct{SentFrom: localIP, SendTo: extBackup.SentFrom, Event: EventAccBackup}
				}
				// Update last time backup was received
				knownElevators[extBackup.SentFrom].Time = time.Now()
				
			case EventBackupAtAllConfirmed:
				printDebug("BackupConfirmed")
				backup := waitingBackups[extBackup.SentFrom]
				knownElevators[extBackup.SentFrom].State = backup.CurrentState
				knownElevators[extBackup.SentFrom].Time = time.Now()
				haveBackups[extBackup.SentFrom] = true
				delete(waitingBackups, extBackup.SentFrom)
				
			// Backup sender side
			case EventAccBackup:
				// Someone accknowledged our backup
				backupAccknowledgementList[extBackup.SentFrom] = true
				if allOtherAccBackup(backupAccknowledgementList, activeElevators) {
					printDebug("Everyone have acked")
					// Everyone has received backup
					stateChanged = false
					sendBackupChannel <- ExtBackupStruct{SentFrom: localIP, Event: EventBackupAtAllConfirmed}
				}
				
			// Backup Requests
			case EventRequestStateFromElevator:
			//  SendAnswer to backup request
				if have,ok := haveBackups[extBackup.SentFrom]; ok && have {
					printDebug("Pushing backup to" + extBackup.SentFrom)
					sendBackupChannel <- ExtBackupStruct{Event: EventAnsweringBackupRequest,
															SendTo: extBackup.SentFrom,
															SentFrom: localIP,
															BackupData: BackupStruct{CurrentState: knownElevators[extBackup.SentFrom].State,},
															}
				} else {
					printDebug("Dont have backup on that one..")
				}
			}
					
		// Timers
		case <- confirmOnlineTicker.C:
			if stateChanged {
				sendBackupChannel <- ExtBackupStruct{Event: EventSendBackupToAll,
													 SentFrom: localIP,
													 BackupData: BackupStruct{CurrentState: knownElevators[localIP].State,
													 },
													}
				stateChanged = false
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
					assignedIP, err := costFunction.CalculateRespondingElevator(knownElevators, activeElevators, localIP, order.Type, order.Floor)
					order := dispatchedOrders[orderID]
					order.OrderID = locOrderID
					locOrderID = locOrderID + 1
					delete(dispatchedOrders, orderID)
					if err != nil {
						log.Fatal(err)
					} else {
						if assignedIP == localIP{
							editOrderOnThisElevator(order.Floor, order.Type, true, knownElevators, localIP)
							log.Println(knownElevators[localIP].MakeQueue())
							if knownElevators[localIP].GetDirection() == DIR_STOP && !knownElevators[localIP].DoorOpen(){
								startMoving(knownElevators, localIP)
							}
							stateChanged = true
						} else {
							dispatchOrder(*order,assignedIP, localIP, dispatchedOrders, sendOrderChannel)
						}
					}
				}
			}
			
		case <- doorTimer.C:
			printDebug("Closing door ")
			knownElevators[localIP].State.OpenDoor = false
			hardwareSim.SetLights(DOOR_LAMP, 0, false)

			// Check if we should start to move
			if knownElevators[localIP].State.HaveOrders(){
				startMoving(knownElevators, localIP)
			} else {
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
			activeElevators[key] = false // dont want to see ourself
		}else if time.Since(knownElevators[key].Time) <= timeoutLimit {
			if activeElevators[key] != true {
				activeElevators[key] = true
				printDebug("Added active elevator " + knownElevators[key].State.LocalIP)
			}
		} else {
			if activeElevators[key] == true {
				printDebug("Removing inactive elevator " + knownElevators[key].State.LocalIP)
				activeElevators[key] = false
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

func dispatchOrder(order OrderStruct, assignedIP, localIP string, dispatchedOrders map[int]*OrderStruct, sendOrderChannel chan ExtOrderStruct){
	dispatchedOrders[order.OrderID] = &order
	dispatchedOrders[order.OrderID].DispatchedTime = time.Now()
	sendOrderChannel <- ExtOrderStruct{SendTo: assignedIP, Order: order, SentFrom: localIP, Event: EventSendOrderToElevator,}
}

func editOrderOnThisElevator(floor, typ int, add bool, knownElevators map[string]*Elevator, localIP string){
	switch typ {
		case BUTTON_CALL_DOWN:
			knownElevators[localIP].State.ExternalOrders[0][floor] = add
		case BUTTON_CALL_UP:
			knownElevators[localIP].State.ExternalOrders[1][floor] = add
	}
	hardwareSim.SetLights(typ, floor, add)
}

func allOtherAccBackup(backupAcknowledgementList, activeElevators map[string]bool) bool {
	for elevatorIP, active := range activeElevators{
		if active{
			if !backupAcknowledgementList[elevatorIP]{
				printDebug("Not enough acks")
				return false
			}
		}
	}
	return true
}

func otherActiveElevators(activeElevators map[string]bool, localIP string) bool {
	for elevatorIp, active := range activeElevators{
		if elevatorIp != localIP && active{
			return true
		}
	}
	return false
}

func openDoor(doorTimer *time.Timer, doorOpentime time.Duration, knownElevators map[string]*Elevator, localIP string){
	doorTimer.Reset(doorOpentime)
	hardwareSim.SetLights(DOOR_LAMP, 0, true)
	knownElevators[localIP].State.OpenDoor = true
}

func startMoving(knownElevators map[string]*Elevator, localIP string){
	knownElevators[localIP].SetDirection(knownElevators[localIP].GetNextDirection())
	knownElevators[localIP].SetMoving(knownElevators[localIP].State.CurrentDirection != DIR_STOP)
	hardwareSim.SetLights(BUTTON_COMMAND, knownElevators[localIP].State.PrevFloor, false)
	hardwareSim.SetMotorDirection(knownElevators[localIP].State.CurrentDirection)
}

func stopMoving(knownElevators map[string]*Elevator, localIP string, floor int){
	hardwareSim.SetMotorDirection(DIR_STOP)
	knownElevators[localIP].SetMoving(false)
	knownElevators[localIP].SetDirection(DIR_STOP)
	hardwareSim.SetLights(BUTTON_COMMAND, floor, false)
}

/*
	Helper function for debuggin
*/
func printDebug(message string){
	if debug{
		log.Println("Main:\t" + message)
	}
}
