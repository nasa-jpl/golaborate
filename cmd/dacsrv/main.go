package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/nasa-jpl/golaborate/acromag"
	"github.com/nasa-jpl/golaborate/generichttp/daq"
	"github.com/nasa-jpl/golaborate/server/middleware/locker"
)

var (
	channels = []int{0, 1, 2, 3, 4, 5}
)

// SetupAP235 initializes the AP235 hardware to a pre-configured and safe condition
func SetupAP235() (*acromag.AP235, error) {
	dac, err := acromag.NewAP235(0)
	if err != nil {
		return dac, err
	}
	for _, ch := range channels {
		err = dac.SetClearVoltage(ch, acromag.MidScale)
		if err != nil {
			return dac, err
		}
		err = dac.SetPowerUpVoltage(ch, acromag.MidScale)
		if err != nil {
			return dac, err
		}
		err = dac.SetRange(ch, "-10,10")
		if err != nil {
			return dac, err
		}
		err = dac.SetOverRange(ch, false)
		if err != nil {
			return dac, err
		}

		err = dac.SetOutputSimultaneous(ch, false)
		if err != nil {
			return dac, err
		}

		// this means output glitches if the FIFO is emptied
		// instead of playback stopping
		err = dac.SetClearOnUnderflow(ch, false)
		if err != nil {
			return dac, err
		}

		// lastly, power up the DAC channel
		err = dac.Output(ch, 0)
		if err != nil {
			return dac, err
		}
	}

	ch2 := []int{0, 1, 2} // JM channels, special bootup
	dac.SetTriggerDirection(false)
	for _, ch := range ch2 {
		dac.SetTriggerMode(ch, "timer")
		dac.SetClearOnUnderflow(ch, true)
	}
	return dac, err
}

// SetupAP236 initializes the AP236 hardware to a pre-configured and safe condition
func SetupAP236() (*acromag.AP236, error) {
	dac, err := acromag.NewAP236(0)
	if err != nil {
		return dac, err
	}
	for _, ch := range channels {
		err = dac.SetClearVoltage(ch, acromag.MidScale)
		if err != nil {
			return dac, err
		}
		err = dac.SetPowerUpVoltage(ch, acromag.MidScale)
		if err != nil {
			return dac, err
		}
		err = dac.SetRange(ch, "-10,10")
		if err != nil {
			return dac, err
		}
		err = dac.SetOverRange(ch, false)
		if err != nil {
			return dac, err
		}

		err = dac.SetOutputSimultaneous(ch, false)
		if err != nil {
			return dac, err
		}

		// lastly, power up the DAC channel
		err = dac.Output(ch, 0)
		if err != nil {
			return dac, err
		}
	}
	return dac, err
}

// SetupHTTP creates a new chi router that exposes an interface to the DAC
func SetupHTTP(dac daq.DAC) chi.Router {
	httpD := daq.NewHTTPDAC(dac)
	lock := locker.New()
	locker.Inject(httpD, lock)
	r := chi.NewRouter()
	httpD.RouteTable.Bind(r)
	return r
}

// LoadWaveform loads a waveform into system memory, for on-demand copying into
// the DAC's buffer
func LoadWaveform(dac *acromag.AP235, name string, period time.Duration) error {
	if period < 0 {
		return errors.New("LoadWaveform: period must be positive")
	}
	f, err := os.Open(name)
	if err != nil {
		return err
	}
	defer f.Close()
	periodNano := period.Nanoseconds()
	err = daq.LoadCSVFloats(dac, f, uint32(periodNano))
	if err != nil {
		return err
	}
	return nil
}

func main() {
	root := chi.NewRouter()
	root.Use(middleware.Logger)
	log.Println("connecting to AP235 (waveform DAC).  If the program is hanging, the driver has glitched;\n reboot the computer")
	ap235, err := SetupAP235()
	if err != nil {
		log.Println("Error configuring AP235, hardware may be missing; remote access to AP235 will not be configured", err)
	} else {
		r235 := SetupHTTP(ap235)
		root.Mount("/ap235/", r235)
		r235.Post("/load-waveform", func(w http.ResponseWriter, r *http.Request) {
			type msg struct {
				Filename string `json:"filename"`
				Periodns int64  `json:"period_ns"`
			}
			var input msg
			err := json.NewDecoder(r.Body).Decode(&input)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			err = LoadWaveform(ap235, input.Filename, time.Duration(input.Periodns)*time.Nanosecond)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
		})
		log.Println("AP235 available via HTTP at /ap235")
	}
	log.Println("connecting to AP236 (non-waveform DAC).  If the program is hanging, the driver has glitched;\n reboot the computer")
	ap236, err := SetupAP236()
	if err != nil {
		log.Println("Error configuring AP236, hardware may be missing; remote access to AP236 will not be configured", err)
	} else {
		r236 := SetupHTTP(ap236)
		root.Mount("/ap236/", r236)
		log.Println("AP236 available via HTTP at /ap236")
	}
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGABRT, syscall.SIGTERM)
	go func() {
		<-ch
		if ap235 != nil {
			ap235.Close()
		}
		if ap236 != nil {
			ap236.Close()
		}
		os.Exit(0)
	}()
	log.Fatal(http.ListenAndServe(":8000", root))
}
