package udp

/*
	This module implements the UDP protocol for our project, or something..
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

type UDPMessage struct {
	ReturnAddress string // "Broadcast" or some ip address.
	Data          []byte
	Length        int // Lengt of the received data, in bytes. N/A for sending.
}

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

	// Generate local address
	tempConnection, err := net.DialUDP(udp4, nil, broadcastAdress)
	if err != nil {
		log.Println("UDP:\t No network connection")
		return "", err
	} else {
		defer tempConnection.Close()
	}
	tempAddress := tempConnection.LocalAddr()
	localAddress, err = net.ResolveUDPAddr(udp4, tempAddress.String())
	if err != nil {
		log.Println("UDP:\t Could not resolve local address.")
		return "", err
	} else if debug {
		log.Printf("UDP:\t Generating local address: \t%s \n", localAddress.String())
	}
	localAddress.Port = localListenPort

	// Create local listening connections
	localListenConnection, err := net.ListenUDP(udp4, localAddress)
	if err != nil {
		log.Println("UDP:\t Couldn't create a UDP listener socket.")
		return "", err
	} else if debug {
		log.Println("UDP:\t Created a UDP listener socket.")
	}

	// Create a listener on broadcast connection.
	broadcastListenConnection, err := net.ListenUDP(udp4, &net.UDPAddr{IP: net.IPv4(0, 0, 0, 0), Port: broadcastListenPort})
	if err != nil {
		log.Println("UDP:\t Could not create a UDP broadcast listen socket.")
		localListenConnection.Close()
		return "", err
	} else if debug {
		log.Println("UDP:\t Created a UDP broadcast listen socket.")
	}
	go udpReceiveServer(localListenConnection, broadcastListenConnection, messageSize, receiveChannel)
	go udpTransmittServer(localListenConnection, broadcastListenConnection, localListenPort, broadcastListenPort, sendChannel)
	return localAddress.IP.String(), err
}

func udpTransmittServer(localConnection, broadcastConnection *net.UDPConn, localListenPort, broadcastListenPort int, sendChannel <-chan UDPMessage) {
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
		select {
		case msg := <-sendChannel:
			if debug {
				log.Println("UDPTransmitServer:\t Start sending an ElevState package to: ", msg.ReturnAddress)
				log.Println("UDP-Send:\t", string(msg.Data))
			}
			if msg.ReturnAddress == "broadcast" {
				bytesSended, err := localConnection.WriteToUDP(msg.Data, broadcastAdress)
				if (err != nil || bytesSended < 0) && debug {
					log.Println("UDPTransmitServer:\t Error ending broadcast message.")
					log.Println(err)
				}
			} else {
				returnAddress, err := net.ResolveUDPAddr("udp", msg.ReturnAddress+":"+strconv.Itoa(localListenPort))
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
	go udpConnectionReader(localConnection, messageSize, localConnectionReceiveChannel)
	go udpConnectionReader(broadcastConnection, messageSize, broadcastConnectionReceiveChannel)
	for {
		select {
		case message := <-broadcastConnectionReceiveChannel:
			receiveChannel <- message
		case message := <-localConnectionReceiveChannel:
			receiveChannel <- message
		}
	}
}

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
			receiveChannel <- UDPMessage{ReturnAddress: returnAddress.String(), Data: buffer[:n], Length: n}
		}
	}
}
