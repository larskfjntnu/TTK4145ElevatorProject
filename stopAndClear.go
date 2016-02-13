package main

import(
	"hardware"
	"time"
	"driver"
)
func main() {
	driver.IOInit()
	time.Sleep(time.Second*1)
	hardware.ResetLights()
	hardware.SetMotorDirection(0)
	time.Sleep(time.Second*1)
}