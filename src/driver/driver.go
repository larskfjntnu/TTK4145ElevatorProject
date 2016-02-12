package driver

/*
#cgo LDFLAGS: -lpthread -lcomedi -lm
#include "io.h"
*/
import "C"
import "errors"

func IOInit() error {
	if err := int(C.io_init(0)); err == 0 {
		return errors.New("Could not initialize hardware.")
	}
	return nil
}

func IOSetBit(channel int) {
	C.io_set_bit(C.int(channel))
}
func IOClearBit(channel int) {
	C.io_clear_bit(C.int(channel))
}
func IOReadBit(channel int) {
	return bool(int(C.io_read_bit(C.int(channel))) != 0)
}

func IOWriteAnalog(channel, value int) {
	C.io_write_analog(C.int(channel), C.int(value))
}

func IOReadAnalog(channel int) int {
	return int(C.io_read_analog(C.int(channel)))
}
