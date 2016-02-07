package network

import (
	"encoding/json"
	"log"

	. "../typedef"
	"../udp"
)

const debug = false

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
				UDPSendChannel <- udp.UDPMessage{ReturnAddress: "broadcast", Data: networkPacket}
				temp := "Sent a message with content: " + message.Message
				printDebug(temp)
			}
		}
	}
}

func printDebug(s string) {
	if debug {
		log.Println("NETWORK:\t", s)
	}
}
