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
	. "../typedef"
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
func Init(receiveChannel chan<- SomeStructToPassOnNetwork,
	sendChannel <-chan SomeStructToPassOnNetwork) (localIP string, err error) {
	const messageSize = 4 * 1024
	const UDPLocalListenPort = 22301
	const UDPBroadcastListenPort = 22302
	UDPSendChannel := make(chan udp.UDPMessage, 10)
	UDPReceiveChannel := make(chan udp.UDPMessage)
	localIP, err = udp.Init(UDPLocalListenPort, UDPBroadcastListenPort, messageSize, UDPSendChannel, UDPReceiveChannel)
	if err != nil {
		return "", err
	}
	go receiveMessageHandler(receiveChannel, UDPReceiveChannel)
	go sendMessageHandler(sendChannel, UDPSendChannel)
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
func receiveMessageHandler(receiveChannel chan<- SomeStructToPassOnNetwork, UDPReceiveChannel <-chan udp.UDPMessage) {
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
				message := string(m["Message"].(string))
				log.Println("ReceiveMessageHandler:\t ", message, " from ", string(m["SenderIp"].(string)))
			}
		}
	}
}
/*
	This handle takes care of sending messages on the connectionport or broadcastport.
	It takes in the sendChannel initialized in the calling module and the UDPSendChannel declared in 
	this modules Init function. We send the message in JSON format and prints an error if the marshaling failed.
*/
func sendMessageHandler(sendChannel <-chan SomeStructToPassOnNetwork, UDPSendChannel chan<- udp.UDPMessage) {
	for {
		log.Println("Network:\t SendmessageHandler")
		select {
		case message := <-sendChannel:
			networkPacket, err := json.Marshal(message)
			if err != nil {
				printDebug("Error Marshalling an outgoing message")
				log.Println(err)
			} else {
				UDPSendChannel <- udp.UDPMessage{RAddress: "broadcast", Data: networkPacket}
				temp := "Sent a message with content: " + message.Message
				printDebug(temp)
			}
		}
	}
}

/*
	Helper function that prints to console if the program is in debug mode.
*/
func printDebug(s string) {
	if debug {
		log.Println("NETWORK:\t", s)
	}
}
