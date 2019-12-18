package main

import (
	"fmt"
	"log"
	"time"

	"github.com/google/gousb"
)

const (
	// TLVID is the ThorLabs vendor ID
	TLVID = 0x1313

	// LDC4001PID is the Product ID for the LDC4001 laser diode / TEC controller
	LDC4001PID = 0x804a
)

func main() {
	ctx := gousb.NewContext()
	defer ctx.Close()
	dev, err := ctx.OpenDeviceWithVIDPID(TLVID, LDC4001PID)
	if err != nil {
		log.Fatal("open", err)
	}
	err = dev.Reset()
	if err != nil {
		log.Fatal("reset", err)
	}
	err = dev.SetAutoDetach(true)
	if err != nil {
		log.Fatal("set auto detach", err)
	}
	defer dev.Close()
	intf, done, err := dev.DefaultInterface()
	if err != nil {
		log.Fatal("iface", err)
	}
	defer done()
	outp, err := intf.OutEndpoint(2)
	if err != nil {
		log.Fatal("outpt", err)
	}
	inp, err := intf.InEndpoint(2)
	if err != nil {
		log.Fatal("endpt", err)
	}
	fmt.Println(outp.String())
	offM := append([]byte("OUTPUT OFF"), '\n')
	onM := append([]byte("OUTPUT ON"), '\n')
	recv := make([]byte, 64)
	for idx := 0; idx < 10; idx++ {
		fmt.Println("about to write")
		// set the diode state on and see the response
		n, err := outp.Write(onM)
		if err != nil || n != len(onM) {
			log.Fatal(err)
		}
		fmt.Println("wrote", n, "bytes")
		time.Sleep(3 * time.Second)
		fmt.Println("about to read")
		n, err = inp.Read(recv)
		if err != nil || n == 0 {
			log.Fatal(err)
		}
		resp := recv[:n]
		fmt.Println(string(resp))

		// set the diode state off and see the response
		n, err = outp.Write(onM)
		if err != nil || n != len(offM) {
			log.Fatal(err)
		}
		time.Sleep(3 * time.Second)
		n, err = inp.Read(recv)
		if err != nil || n == 0 {
			log.Fatal(err)
		}
		resp = recv[:n]
		fmt.Println(string(resp))

	}
}
