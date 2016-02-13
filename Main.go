package somepackage

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
	"./src/udp"
)
const sending = false

// const debug = true
// const broadcast = true

// // This function starts the program. The while loop inside runs throughout
// // the lifetime of the program.
// func main() {
// 	// Do some initialization. Determine wether Master or slave/client.
// 	runtime.GOMAXPROCS(runtime.NumCPU())
// 	const connectionAttempsLimit = 10
// 	const iAmAliveTickTime = 100 * time.Millisecond
// 	const ackTimeout = 500 * time.Millisecond

// 	//	------------- Initialize network ----------------
// 	receiveChannel := make(chan typedef.SomeStructToPassOnNetwork, 5)
// 	sendChannel := make(chan typedef.SomeStructToPassOnNetwork)
// 	localIP, err := initNetwork(connectionAttempsLimit, receiveChannel, sendChannel)
// 	if err != nil {
// 		log.Println("MAIN:\t Network init failed")
// 		log.Fatal(err)
// 	} else if debug {
// 		log.Println("MAIN:\t Network init succesful")
// 		log.Printf("MAIN:\t LocalIp:\t %s\n", localIP)
// 	}
// 	if broadcast {
// 		sendChannel <- typedef.SomeStructToPassOnNetwork{
// 			Message:  "Hi, i'm broadcasting!",
// 			SenderIp: localIP,
// 		}
// 	}

// 	// This is the main loop running continuously.
// 	for {
// 		// TODO -> Implement the main logic of the elevator.
// 		select {
// 		case message := <-receiveChannel:
// 			log.Println("MAIN:\t Received message!")
// 			log.Println(message.Message)
// 		}
// 	}
// }

// func initNetwork(connectionAttempsLimit int, receiveChannel, sendChannel chan typedef.SomeStructToPassOnNetwork) (localIP string, err error) {
// 	for i := 0; i <= connectionAttempsLimit; i++ {
// 		localIP, err := network.Init(receiveChannel, sendChannel)
// 		if err != nil {
// 			if i == 0 {
// 				log.Println("MAIN:\t Network init was not successfull. Trying some more times.")
// 			} else if i == connectionAttempsLimit {
// 				return "", err
// 			}
// 			time.Sleep(3 * time.Second)
// 		} else {
// 			return localIP, nil
// 		}
// 	}
// 	return "", nil
// }

func print_udp_message(msg udp.UDPMessage){
	fmt.Printf("msg:  \n \t raddr = %s \n \t data = %s \n \t length = %v \n", msg.RAddress, msg.Data, msg.Length)
}

func node (send_ch, receive_ch chan udp.UDPMessage){
for {
	time.Sleep(1*time.Second)
	snd_msg := udp.UDPMessage{RAddress:"broadcast", Data:[]byte("Hello World"), Length:11}
	if sending {
		fmt.Printf("Sending------\n")
		send_ch <- snd_msg
		print_udp_message(snd_msg)
	}
	fmt.Printf("Receiving----\n")
	rcv_msg:= <- receive_ch
	print_udp_message(rcv_msg)
	}
}


func main (){
	send_ch := make (chan udp.UDPMessage)
	receive_ch := make (chan udp.UDPMessage)
	_, err := udp.Init(20001, 20002, 1024, send_ch, receive_ch)	
	go node(send_ch, receive_ch)


	if (err != nil){
		fmt.Print("main done. err = %s \n", err)
	}
		neverReturn := make (chan int)
	<-neverReturn

}
