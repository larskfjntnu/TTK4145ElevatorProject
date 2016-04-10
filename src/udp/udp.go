package udp

/*
	This module is used for UDP communicating with the network. It is used by the Network module.
	It sets the IP for boradcasting and ports for connections and broadcasting.
	It expects to receive JSON serialized objects to send and it outputs JSON serialized objects
	when they are received.
*/

import (
	"net"
	"strconv"
	"log"
)

const debug = true

var locAddr *net.UDPAddr
var bcAddr *net.UDPAddr

type UDPMessage struct{
	RAddress string // "broadcast" or specific ip.
	Data []byte
	Length int // Length of the received data packet, number of bytes, nil for sending
}

/*
	This function initialized the  broadcast-  and local connection. It uses a temporary connection to resolve the
	local address of the host. It then starts to go routines acting as servers handling receving transmissions
	and sending messages respectively. It returns the local IP address of the host, or an error message
	if any. It takes in all the necessary channels (send and receive) from its calling routine, enabling
	interaction between the servers and other routines in the program.
*/
func Init(localListenPort, broadcastListenPort, messageSize int, sendChannel <- chan UDPMessage, receiveChannel chan<- UDPMessage) (localIp string, err error){
	// Generate broadcast address
	bcAddr, err := net.ResolveUDPAddr("udp4", "255.255.255.255:" + strconv.Itoa(broadcastListenPort))
	if err != nil {
		printDebug("Error resolving UDPAddress")
		return "", err
	} else {
		printDebug("Generated broadcast address: " + bcAddr.String())
	}

	// Generate localaddress
	tempCon, err := net.DialUDP("udp4", nil, bcAddr)
	if err != nil {
		printDebug("No network connection.")
		return "", err
	} else {
		defer tempCon.Close()
	}

	tempAddr := tempCon.LocalAddr()
	locAddr, err := net.ResolveUDPAddr("udp4", tempAddr.String())
	if err != nil {
		printDebug("Could not resolve local address")
		return "", err
	} else {
		printDebug("Generated local address: " + locAddr.String())
	}
	locAddr.Port = localListenPort


	// Generate local listening connections.
	localListenCon, err := net.ListenUDP("udp4", locAddr)
	if err != nil {
		printDebug("Could not create a UDP listener socket")
		return "", err
	} else {
		printDebug("Create UDP listener socket")
	}

	// Generate listener on broadcast connection
	bcListenCon, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(0, 0, 0, 0), Port: broadcastListenPort})
	if err != nil {
		printDebug("Could not create a UDP broadcast listen socket")
		localListenCon.Close()
		return "", err
	} else {
		printDebug("Created a UDP broadcast listen socket")
	}

	go udpReceiveServer(localListenCon, bcListenCon, messageSize, receiveChannel)
	go udpTransmitServer(localListenCon, bcListenCon, localListenPort, broadcastListenPort, sendChannel)
	return locAddr.IP.String(), err
}

/*
	This functions is run as a go routine acting as a server for handling the sending/transmission
	of messages in the form of an UDPMessage(struct). It communicates with other routines through a 
	channel(sendChannel) where it receives UDPMessages to send.
*/
func udpTransmitServer(loccon, bccon *net.UDPConn, localListenPort, bcListenPort int, sendChannel <-chan UDPMessage) {
	defer func() {
		if r := recover(); r != nil {
			printDebug("Error in udpTransmitServer: " + "\nClosing connection")
			loccon.Close()
			bccon.Close()
		}
	}()

	for {
		select{
		case message := <- sendChannel:
			printDebug("TransmitServer :\t Start sending a state package to: " + message.RAddress)
			printDebug("Send: \t" + string(message.Data))

			if message.RAddress == "broadcast" {
				n, err := loccon.WriteToUDP(message.Data, bcAddr)
				if (err != nil || n < 0) {
					printDebug("Error ending broadcast message")
					log.Println(err)
				}
			} else {
				raddr, err := net.ResolveUDPAddr("udp4", message.RAddress + ":" + strconv.Itoa(localListenPort))
				if err != nil {
					printDebug("TransmitServer:\t Could not resolve RAddress")
					log.Fatal(err)
				}
				if n, err := loccon.WriteToUDP(message.Data, raddr); err != nil || n < 0 {
					printDebug("TransmitServer:\t Error sending p2p message")
					log.Println(err)
				}
			}
		}
	}
}

/*
	This functions is run as a go routine acting as a server for handling the receiving
	of messages in the form of an UDPMessage(struct). It communicates with other routines through a 
	channel(receiveChannel) where it sends UDPMessages it has received. It starts two goroutines
	responsible for handling incoming messages through broadcasts and p2p respectively. The receiveServer
	communicates with these routines through two locally created channels where UDPMessages are being sent.
*/

func udpReceiveServer(loccon, bccon *net.UDPConn, messageSize int, receiveChannel chan<- UDPMessage) {
	defer func(){
		if r := recover(); r != nil {
			printDebug("Error in udpReceiveServer: " +  " Closing connection")
			loccon.Close()
			bccon.Close()
		}
	}()

	bcconRcvCh := make(chan UDPMessage)
	locconRcvCh := make(chan UDPMessage)
	go udpConnectionReader(loccon, messageSize, locconRcvCh)
	go udpConnectionReader(bccon, messageSize, bcconRcvCh)

	for{
		select{
		case message := <- bcconRcvCh:
			receiveChannel <- message
		case message := <- locconRcvCh:
			receiveChannel <- message
		}
	}
}

/*
	This function is run as a goroutine reading incoming connections on the given connection(conn).
	It verifies that the received message is valid before passing it to the calling routine through a channel(rcvCh)
	where the message is passed as a UDPMessage.
*/

func udpConnectionReader(conn *net.UDPConn, messageSize int, rcvCh chan<- UDPMessage) {
	defer func(){
		if r := recover(); r != nil {
			printDebug("ConnectionReader:\t Error in connectionReader: " +  "\nClosing connection")
			conn.Close()
		} 
	}()

	for {
		buffer := make([]byte, messageSize)
		n, raddr, err := conn.ReadFromUDP(buffer)
		if err != nil || n < 0 || n > messageSize {
			printDebug("Error in ReadFromUDP" + err.Error())
		} else {
			printDebug("ConnectionReader:\t Received package from: " + raddr.String())
			printDebug("Listen:\t" + string(buffer[:]))
			rcvCh <- UDPMessage{RAddress: raddr.String(), Data: buffer[:n], Length: n}
		}
	}
}


/*
	Helper function for debug messages.
*/
func printDebug(message string){
	if debug{
		log.Println("UDP:\t" + message)
	}
}
