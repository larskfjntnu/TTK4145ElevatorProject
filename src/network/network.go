package network
/*
	This is the network module which takes care of sending and receiving messages or broadcasts.
	It communicates with other modules through two channels, sendChannel and receiveChannel, which holds the
	serialized object to send and that wich are received. It uses the UDP protocol to communicate on the 
	network, and acknowledgement is done on application level.
	Messages are expected to be Structs which can be zerialized to JSON.
	It uses the UDP module to do the actual networking on the UDP protocol.
*/

import (
	"encoding/json"
	"log"
	."../typedef"
	"../udp"
)

// Constant used to determine output to console-
const debug = false

/* 
	This function initializes the network module, based on the channel passed from the calling module.
	It returns this systems/modules ip on the local network or, if any, error. 
	It sets the hardcoded ports for listening and broadcast ports, and makes two corresponding UDPMessage channels
	for sending and receiving. It calls the init function from the udp module to set up the udp connection.
*/
func Init(receiveOrderChannel chan ExtOrderStruct, receiveChannel chan ExtBackupStruct,
	sendOrderChannel chan ExtOrderStruct, sendChannel chan ExtBackupStruct) (localIP string, err error) {
	const messageSize = 4 * 1024
	const UDPLocalListenPort = 22301
	const UDPBroadcastListenPort = 22302
	UDPSendChannel := make(chan udp.UDPMessage, 10)
	UDPReceiveChannel := make(chan udp.UDPMessage)
	localIP, err = udp.Init(UDPLocalListenPort, UDPBroadcastListenPort, messageSize, UDPSendChannel, UDPReceiveChannel)
	if err != nil {
		return "", err
	}
	go receiveOrderAndBackupServer(receiveOrderChannel, receiveChannel, UDPReceiveChannel)
	go sendOrderAndBackupHandler(sendOrderChannel, sendChannel, UDPSendChannel)
	printDebug("Network initialized")
	return localIP, nil
}

/*
	This handle takes care of received messages on the connectionport or broadcastport.
	It takes in the receiveChannel initialized in the calling module and the UDPReceiveChannel declared in 
	this modules Init function. We send the message in JSON format and prints an error if the unmarshaling failed.
	Per 7/2-16 it prints the message directly to console, but in elevator context it should pass the received message
	to the main module.(Make generic interface which supports extracting received messages.)
	The receive channel is where these messages are passed to the calling module.
*/
func receiveOrderAndBackupServer(receiveOrderChannel chan ExtOrderStruct, receiveBackupChannel chan ExtBackupStruct, UDPReceiveChannel <-chan udp.UDPMessage) {
	for {
		select {
		case message := <-UDPReceiveChannel:
			var f interface{}
			err := json.Unmarshal(message.Data[:message.Length], &f)
			if err != nil {
				printDebug("Error with Unmarshaling a message.")
				log.Println(err)
			} else {
				m := f.(map[string]interface{})
				eventNum := int(m["Event"].(float64))
				if eventNum <= 2 && eventNum >= 0 {
					var newExtOrder = ExtOrderStruct{}
					if err := json.Unmarshal(message.Data[:message.Length], &newExtOrder); err == nil{
						if newExtOrder.Valid() {
							receiveOrderChannel <- newExtOrder;
							printDebug("Received a ExtOrderStruct with event: " + EventType[newExtOrder.Event])
						} else {
							printDebug("Rejected a ExtOrderStruct with event: " + EventType[newExtOrder.Event])
						}
					} else {
						printDebug("Error unmarshaling an ExtOrderStruct")
					}
				} else if eventNum >= 3 && eventNum <= 4 {
					var newExtBackup = ExtBackupStruct{}
					if err := json.Unmarshal(message.Data[:message.Length], &newExtBackup); err == nil {
						if newExtBackup.Valid() {
							receiveBackupChannel <- newExtBackup
							printDebug("Received an ExtBackupStruct with Event: " + EventType[newExtBackup.Event])
						} else {
							printDebug("Rejected an ExtBackupStruct with Event: " + EventType[newExtBackup.Event])
						}
					} else {
						printDebug("Error unmarshaling a ExtBackupStruct")
					}
				} else {
					log.Println("Received message with unknown type ")
				}				
			}
		}
	}
}

/*
	This handle takes care of sending messages on the connectionport or broadcastport.
	It takes in the sendChannel initialized in the calling module and the UDPSendChannel declared in 
	this modules Init function. We send the message in JSON format and prints an error if the marshaling failed.
*/
func sendOrderAndBackupHandler(sendOrderChannel <-chan ExtOrderStruct, sendChannel <-chan ExtBackupStruct, UDPSendChannel chan<- udp.UDPMessage) {
	for {
		select {
		case message := <- sendOrderChannel:
			packet, err := json.Marshal(message)
			if err != nil {
				printDebug("Error marshaling outgoing message")
				log.Println(err)
			} else {
				UDPSendChannel <- udp.UDPMessage{RAddress: "broadcast", Data: packet}
				printDebug("Sent an OrderStruct with event: " + EventType[message.Event])
			}

		case message := <-sendChannel:
			networkPacket, err := json.Marshal(message)
			if err != nil {
				printDebug("Error marshaling outgoing message")
				log.Println(err)
			} else {
				UDPSendChannel <- udp.UDPMessage{RAddress: "broadcast", Data: networkPacket}
				printDebug("Sent a stateStruct with event: " + EventType[message.Event])
			}
		}
	}
}

/*
	Helper function that prints to console if the program is in debug mode.
*/
func printDebug(s string) {
	if debug {
		log.Println("Network:\t", s)
	}
}
