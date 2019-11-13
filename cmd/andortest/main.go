package main

import (
	"fmt"
	"io/ioutil"
	"time"

	"github.jpl.nasa.gov/HCIT/go-hcit/andor/sdk3"
)

func main() {
	err := sdk3.InitializeLibrary()
	if err != nil {
		return

	}
	defer sdk3.FinalizeLibrary()
	ncam, err := sdk3.DeviceCount()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%d cameras\n", ncam)
	swver, err := sdk3.SoftwareVersion()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("version %s\n", swver)

	c, err := sdk3.Open(2)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer c.Close()
	err = sdk3.JPLNeoBootup(c)
	fmt.Println(err)

	c.Allocate()
	c.QueueBuffer()
	height, err := sdk3.GetInt(c.Handle, "AOIHeight")
	fmt.Println(height, err)
	width, err := sdk3.GetInt(c.Handle, "AOIWidth")
	fmt.Println(width, err)
	sdk3.IssueCommand(c.Handle, "AcquisitionStart")
	c.WaitBuffer(1 * time.Second)

	ext := sdk3.ExtCamera{Camera: c}
	// arrptr, err := ext.LastFrame()
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }
	buf, err := ext.BufferCopy()
	if err != nil {
		fmt.Println(err)
		return
	}
	err = ioutil.WriteFile("tmp.bin", buf, 0777)
	if err != nil {
		fmt.Println(err)
	}
	return
}
