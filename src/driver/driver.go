package driver

/*
	Need to implement the driver source. These files are C code. 
	Go recognices the 'import' statement within the comment and lets
	us reference the functions in the interface of the C code in the
	Go source code. The 'import "C" ' statement is a 'pseudo package' which
	let cgo recognise the C namespace. 
	The 'import "unsafe" ' is needed because the memory allocations made by
	C are not known to the Go memory manager. When C creates a string or such,
	we need to free this by calling C.free
*/


/*
#cgo LDFLAGS: -lpthread -lcomedi -lcomedi -lm
#cgo CFLAGS: -std=c99
#include "io.h"
*/
import "C"
import "errors"

func IOInit() error {
	if err := int(C.io_init()); err == 0 {
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
func IOReadBit(channel int) bool{
	return bool(int(C.io_read_bit(C.int(channel))) != 0)
}

func IOWriteAnalog(channel, value int) {
	C.io_write_analog(C.int(channel), C.int(value))
}

func IOReadAnalog(channel int) int {
	return int(C.io_read_analog(C.int(channel)))
}
