package main

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"time"

	"github.jpl.nasa.gov/bdube/golab/acromag/ap236"
)

func main() {
	reader := bufio.NewReader(os.Stdin)
	log.Println("connecting to AP236 #0")
	dac, err := ap236.New(0)
	if err != nil {
		log.Fatal(err)
	}
	defer dac.Close()
	log.Println("Enter a channel number:")
	str, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}
	channel, err := strconv.Atoi(str[:len(str)-1])
	if err != nil {
		log.Fatal(err)
	}
	log.Println("setting range to +/- 10V")
	dac.SetRange(channel, "-10,10")
	log.Println("setting overrange false")
	dac.SetOverRange(channel, false)
	log.Println("setting thermal protection (shut down overtemp) true")
	dac.SetOverTempBehavior(channel, true)

	log.Println("advancing to basic range testing.")
	log.Println("floating point interface:")
	log.Println("press enter to command -10V")
	reader.ReadString('\n')
	err = dac.Output(channel, -10)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("press enter to move to +10V")
	reader.ReadString('\n')
	log.Println("DN interface:")
	log.Println("press enter to command 0 (-10V)")
	reader.ReadString('\n')
	dac.OutputDN16(channel, 0)
	log.Println("press enter to command 65535 (+10V)")
	reader.ReadString('\n')
	dac.OutputDN16(channel, 65535)

	log.Println("advancing to step test")
	log.Println("press enter to reset DAC to -10V")
	reader.ReadString('\n')
	dac.Output(1, -10)
	log.Println("press enter to step to +10V")
	reader.ReadString('\n')
	dac.Output(1, 10)
	time.Sleep(time.Second)
	dac.OutputDN16(channel, 65535/2)

	log.Println("advancing to latency test")
	var start uint16 = 65535 / 2
	up := start + 10000
	log.Println("press enter to command the DAC to move up 10000 DN (~3V) then back down ASAP")
	log.Println("expect ~10us")
	dac.OutputDN16(channel, start)
	dac.OutputDN16(channel, up)
	dac.OutputDN16(channel, start)

	log.Println("advancing to ramp test")
	log.Println("start=0, stop=65535, step=100, dT=15ms, steps=655 (~10s)")
	log.Println("press enter to start")
	reader.ReadString('\n')
	var (
		out  uint16
		stop uint16 = 65535
		step uint16 = 100
		dT          = 15 * time.Millisecond
	)
	for ; out < stop; out += step {
		dac.OutputDN16(channel, out)
		time.Sleep(dT)
	}

	log.Println("test complete")

}
