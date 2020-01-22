package main

import (
	"fmt"
	"log"

	"github.jpl.nasa.gov/HCIT/go-hcit/andor/sdk2"
)

func main() {
	// copied from examples/console/image.cpp
	err := sdk2.Initialize("/usr/local/etc/andor")
	if err != nil {
		log.Fatal(err)
	}

	// single scan acq and image read mode
	c := sdk2.Camera{}
	defer c.ShutDown()
	err = c.SetFan(true)

	if err != nil {
		log.Fatal(err)
	}
	err = c.SetTemperatureSetpoint("0")
	if err != nil {
		log.Fatal(err)
	}
	err = c.SetCoolerActive(true)
	if err != nil {
		log.Fatal(err)
	}

	// err = c.SetVSAmplitude(sdk2.VerticalClockNormal)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// // set HS speed to as slow as possible
	// i, err := c.GetNumberHSSpeeds(0) // 0 -> ADC ch 0
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// err = c.SetHSSpeed(i - 1)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// // set VS speed to as slow as possible
	// i, err = c.GetNumberVSSpeeds()
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// err = c.SetVSSpeed(i - 1)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// err = c.SetReadoutMode(sdk2.ReadoutImage)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// err = c.SetAcquisitionMode(sdk2.AcquisitionSingleScan)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	fmt.Println(c.GetTemperature())

}
