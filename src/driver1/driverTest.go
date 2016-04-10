package driver1

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




func IOInit() error {
	return nil
}

func IOSetBit(channel int) {
}
func IOClearBit(channel int) {
}
func IOReadBit(channel int) bool{
	return false
}

func IOWriteAnalog(channel, value int) {
}

func IOReadAnalog(channel int) int {
	return 1
}