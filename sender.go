package pk2

import (
	"fmt"
	"time"

	"./src/udp"
)

const sending = true

func print_udp_message(msg udp.UDPMessage) {
	fmt.Printf("msg:  \n \t raddr = %s \n \t data = %s \n \t length = %v \n", msg.RAddress, msg.Data, msg.Length)
}

func node(send_ch, receive_ch chan udp.UDPMessage) {
	data := "I would like to connect!"
	rAddr := "broadcast"
	for {
		time.Sleep(1 * time.Second)
		snd_msg := udp.UDPMessage{RAddress: rAddr, Data: []byte(data), Length: len(data)}
		if sending {
			fmt.Printf("Sending------\n")
			send_ch <- snd_msg
			print_udp_message(snd_msg)
		}
		fmt.Printf("Receiving----\n")
		rcv_msg := <-receive_ch
		if string(rcv_msg.Data[:rcv_msg.Length]) == "We are connected!" {
			fmt.Printf("Connected----\n")
			data = "It is so cool that we are connected!"
			rAddr = rcv_msg.RAddress
			snd_msg := udp.UDPMessage{RAddress: rAddr, Data: []byte(data), Length: len(data)}
			send_ch <- snd_msg
		}
		print_udp_message(rcv_msg)
	}
}

func main() {
	send_ch := make(chan udp.UDPMessage)
	receive_ch := make(chan udp.UDPMessage)
	_, err := udp.Init(20001, 20002, 1024, send_ch, receive_ch)
	go node(send_ch, receive_ch)

	if err != nil {
		fmt.Print("main done. err = %s \n", err)
	}
	neverReturn := make(chan int)
	<-neverReturn

}
