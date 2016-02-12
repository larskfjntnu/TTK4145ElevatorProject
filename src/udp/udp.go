package udp

/*
	This module is used for UDP communicating with the network. It is used by the Network module.
	It sets the IP for boradcasting and ports for connections and broadcasting.
	It expects to receive JSON serialized objects to send and it outputs JSON serialized objects
	when they are received.
*/

import (
	"log"
	"net"
	"strconv"
)

const debug = false
const udp4 = "udp4"
const broadcastIp = "255.255.255.255:"

// The adresses used for communicating and broadcast.
var localAddress *net.UDPAddr
var broadcastAdress *net.UDPAddr

// The struct for the messages that are being sendt or received.
type UDPMessage struct {
	RAddress string // "broadcast" or some ip address. Return adress when received(the address to return to), receive address when sending(the address which is receiving.)
	Data     []byte // The data of the sendt or received package(serialized JSON)
	Length   int    // Lengt of the received data, in bytes. N/A for sending.
}

/*
	This function initializes the UDP module. It sets the port for listening, broadcasting, the approved message size and the channels
	for communicating with the calling module. It returns the local IP-address of this system/module, and an error if if it fails(then ip is "").
*/
func Init(localListenPort, broadcastListenPort, messageSize int, sendChannel <-chan UDPMessage, receiveChannel chan<- UDPMessage) (localIP string, err error) {

	// Generate broadcast address
	broadcastAdress, err = net.ResolveUDPAddr(udp4, broadcastIp+strconv.Itoa(broadcastListenPort))
	if err != nil {
		log.Println("UDP:\t Could not resolve UDPAddress.")
		return "", err
	} else if debug {
		// We are in debug mode.
		log.Printf("UDP:\t Generating broadcast address:\t %s \n", broadcastAdress.String())
	}

	// Generate local address, uses the tempConnection to fetch address via the UDP dial.
	tempConnection, err := net.DialUDP(udp4, nil, broadcastAdress)
	if err != nil {
		log.Println("UDP:\t No network connection")
		return "", err
	}
	defer tempConnection.Close() // Makes sure the connection is closed when Init completes.
	tempAddress := tempConnection.LocalAddr()
	localAddress, err = net.ResolveUDPAddr(udp4, tempAddress.String())
	if err != nil {
		log.Println("UDP:\t Could not resolve local address.")
		return "", err
	} else if debug {
		log.Printf("UDP:\t Generating local address: \t%s \n", localAddress.String())
	}
	localAddress.Port = localListenPort // Set the port property of the *net.UDPAddr struct

	// Create local listening connections
	localListenConnection, err := net.ListenUDP(udp4, localAddress) // Listens for incoming UDP packets addressed to localAddress.
	if err != nil {
		log.Println("UDP:\t Couldn't create a UDP listener socket.")
		return "", err
	}
	if debug {
		log.Println("UDP:\t Created a UDP listener socket.")
	}

	// Create a listener on broadcast connection.
	broadcastListenConnection, err := net.ListenUDP(udp4, &net.UDPAddr{IP: net.IPv4(0, 0, 0, 0), Port: broadcastListenPort})
	if err != nil {
		log.Println("UDP:\t Could not create a UDP broadcast listen socket.")
		localListenConnection.Close()
		return "", err
	}
	if debug {
		log.Println("UDP:\t Created a UDP broadcast listen socket.")
	}
	// Start goroutines to handle incoming messages and sending outgoing messages.
	go udpReceiveServer(localListenConnection, broadcastListenConnection, messageSize, receiveChannel)
	go udpTransmitServer(localListenConnection, broadcastListenConnection, localListenPort, broadcastListenPort, sendChannel)
	return localAddress.IP.String(), err
}

/*
	This function is called as a goroutine and acts as a server used for sending UDP packets. It receives the packets to send via
	the sendChannel, and runs an infinite loop waiting for messages to send.
*/
func udpTransmitServer(localConnection, broadcastConnection *net.UDPConn, localListenPort, broadcastListenPort int, sendChannel <-chan UDPMessage) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("UDPConnectionReader:\t Error in UDPTransmitServer: %s \n Closing connection.", r)
			localConnection.Close()
			broadcastConnection.Close()
		}
	}()
	for {
		if debug {
			log.Println("UDPTransmitServer:\t Waiting on new value on sendChannel.")
		}
		select { // Waits for something to happen on the sendChannel
		case msg := <-sendChannel:
			if debug {
				log.Println("UDPTransmitServer:\t Start sending an ElevState package to: ", msg.RAddress)
				log.Println("UDP-Send:\t", string(msg.Data))
			}
			if msg.RAddress == "broadcast" { // Broadcast the message
				bytesSended, err := localConnection.WriteToUDP(msg.Data, broadcastAdress)
				if (err != nil || bytesSended < 0) && debug {
					log.Println("UDPTransmitServer:\t Error ending broadcast message.")
					log.Println(err)
				}
			} else { // Send the message to the localConnection. p2p
				returnAddress, err := net.ResolveUDPAddr("udp", msg.RAddress)
				if err != nil {
					log.Println("UDPTransmitServer:\t Could not resolve return address.")
					log.Fatal(err)
				}
				if n, err := localConnection.WriteToUDP(msg.Data, returnAddress); err != nil || n < 0 {
					log.Printf("UDPTransmiServer:\t Error: Sending p2p message.")
					log.Println(err)
				}
			}
		}
	}
}

/*
	This function is called as a goroutine and acts as a server used for receiving UDP packets. It sends the packets received via
	the receiveChannel. It starts two goroutines used for listening on connections and broadcasts for incoming UDP packets.
	These are then sendt via two different channels to the ReceiveServer, depending on the incoming message type(p2p or broadcast.)
*/
func udpReceiveServer(localConnection, broadcastConnection *net.UDPConn, messageSize int, receiveChannel chan<- UDPMessage) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("UDP:\t ERROR in UDPReceiveServer: %s \n Closing connection.", r)
			localConnection.Close()
			broadcastConnection.Close()
		}
	}()
	broadcastConnectionReceiveChannel := make(chan UDPMessage)
	localConnectionReceiveChannel := make(chan UDPMessage)
	// Run the goroutines.
	go udpConnectionReader(localConnection, messageSize, localConnectionReceiveChannel)
	go udpConnectionReader(broadcastConnection, messageSize, broadcastConnectionReceiveChannel)
	for {
		select { // Wait for messages from the above goroutines.
		case message := <-broadcastConnectionReceiveChannel:
			receiveChannel <- message
		case message := <-localConnectionReceiveChannel:
			receiveChannel <- message
		}
	}
}

/*
	Used to listen for incoming UDP packets on  an given connection. Runs an infinite loop reading from the connection to a buffer.
	When a message is complete, it sends it to to the caller via the receive channel.
*/
func udpConnectionReader(connection *net.UDPConn, messageSize int, receiveChannel chan<- UDPMessage) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("UDPConnectionReader:\t ERROR in udpConnectionReader:\t %s \n Closig connection.", r)
			connection.Close()
		}
	}()

	for {
		if debug {
			log.Printf("UDPConnectionReader:\t Waiting on data from UDPConnection %s\n", connection.LocalAddr().String())
		}
		buffer := make([]byte, messageSize) // TODO: Do without allocation memory each time!
		n, returnAddress, err := connection.ReadFromUDP(buffer)
		if err != nil || n < 0 || n > messageSize {
			log.Println("UDPConnectionReader:\t Error in ReadFromUDP:", err)
		} else {
			if debug {
				log.Println("UDPConnectionReader:\t Received package from:", returnAddress.String())
				log.Println("UDP-Listen:\t", string(buffer[:]))
			}
			receiveChannel <- UDPMessage{RAddress: returnAddress.String(), Data: buffer[:n], Length: n}
		}
	}
}
