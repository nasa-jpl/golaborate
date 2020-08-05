package main

import (
	"bufio"
	"log"
	"math"
	"os"
	"strconv"
	"time"

	"github.jpl.nasa.gov/bdube/golab/acromag/ap235"
)

func main() {
	reader := bufio.NewReader(os.Stdin)
	log.Println("connecting to AP236 in slot 1")
	dac, err := ap235.New(1)
	if err != nil {
		log.Fatal(err)
	}
	defer dac.Close()
	log.Println("Enter a channel number:")
	str, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}
	channel, err := strconv.Atoi(str)
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

	log.Println("advancing to waveform test")
	floats := make([]float64, 100000)
	for i := 0; i < len(floats); i++ {
		floats[i] = float64(i) / math.Pi / 5
	}
	dac.SetTriggerMode(channel, "timer")
	dac.SetTimerPeriod(160000) // 160us/sample
	dac.SetTriggerDirection(true)
	dac.SetOperatingMode(channel, "waveform")
	err = dac.PopulateWaveform(channel, floats)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("length=100,000 samples, period=100 samples")
	log.Println("press enter to start playback for 10 seconds")
	reader.ReadString('\n')
	dac.StartWaveform()
	time.Sleep(10)
	log.Println("test complete")
}
