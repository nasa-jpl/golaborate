/*Package envsrv contains the machinery for an environmental recording server.

It captures a temperature and humidity measurement from a fluke DewK every
<duration> and stores up to N of them to return over HTTP.

*/
package envsrv

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.jpl.nasa.gov/HCIT/go-hcit/cryocon"
	"github.jpl.nasa.gov/HCIT/go-hcit/nkt"

	"github.jpl.nasa.gov/HCIT/go-hcit/granvillephillips"
	"github.jpl.nasa.gov/HCIT/go-hcit/ixllightwave"
	"github.jpl.nasa.gov/HCIT/go-hcit/lesker"
	"github.jpl.nasa.gov/HCIT/go-hcit/newport"
	"github.jpl.nasa.gov/HCIT/go-hcit/server"

	"github.jpl.nasa.gov/HCIT/go-hcit/fluke"

	"github.com/brandondube/ringo"
)

// Envmon is an environmental monitor that stores ring buffers of temperature
// and humidity and can serve the slices over HTTP
type Envmon struct {
	T      ringo.CircleF64
	RH     ringo.CircleF64
	Time   ringo.CircleTime
	ticker *time.Ticker
	stop   chan struct{}
}

type envdata struct {
	T    *[]float64   `json:"temp"`
	RH   *[]float64   `json:"rh"`
	Time *[]time.Time `json:"timestamp"`
}

// New creates a new Envmon and initializes the internal machinery
func New(tick time.Duration, capacity int) Envmon {
	ticker := time.NewTicker(tick)
	T := ringo.CircleF64{}
	T.Init(capacity)
	RH := ringo.CircleF64{}
	RH.Init(capacity)
	Time := ringo.CircleTime{}
	Time.Init(capacity)
	return Envmon{
		T:      T,
		RH:     RH,
		Time:   Time,
		ticker: ticker}
}

// Start triggers operation of the monitor
func (em *Envmon) Start() {
	go em.runner()
}

// Stop kills the monitor.  It may be restarted.
func (em *Envmon) Stop() {
	em.stop <- struct{}{}
}

func (em *Envmon) runner() {
	for {
		select {
		case t := <-em.ticker.C:
			resp, err := http.Get("http://misery.jpl.nasa.gov:8081/zygo-bench/temphumid")
			if err != nil {
				log.Printf("error getting data from Fluke, %q\n", err)
				continue
			}
			th := fluke.TempHumid{}
			err = json.NewDecoder(resp.Body).Decode(&th)
			defer resp.Body.Close()
			if err != nil {
				log.Printf("error decoding JSON data from Fluke, %q\n", err)
				continue
			}
			em.Time.Append(t)
			em.T.Append(th.T)
			em.RH.Append(th.H)
		case <-em.stop:
			return
		}
	}
}

// HTTPYield returns an object over HTTP which contains arrays of temp, humidity, and timestamps
func (em *Envmon) HTTPYield(w http.ResponseWriter, r *http.Request) {
	bufT := em.T.Contiguous()
	bufRH := em.RH.Contiguous()
	bufTime := em.Time.Contiguous()
	s := envdata{
		T:    &bufT,
		RH:   &bufRH,
		Time: &bufTime}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(s)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	return
}

// SetupDevices creates new instances of the various device types
// and packs them into a mainframe object
func SetupDevices(c Config) server.Mainframe {
	mf := server.Mainframe{}
	for _, params := range c.Flukes {
		flk := fluke.NewDewK(params.Addr, params.URLStem, params.Serial)
		mf.Add(flk)
	}
	for _, params := range c.IXLLightwaves {
		diode := ixllightwave.NewLDC3916(params.Addr, params.URLStem)
		mf.Add(diode)
	}
	for _, params := range c.GPConvectrons {
		conv := granvillephillips.NewSensor(params.Addr, params.URLStem, params.Serial)
		mf.Add(conv)
	}
	for _, params := range c.Leskers {
		kjc := lesker.NewSensor(params.Addr, params.URLStem, params.Serial)
		mf.Add(kjc)
	}
	for _, params := range c.ESP300s {
		ctl := newport.NewESP301(params.Addr, params.URLStem, params.Serial)
		mf.Add(ctl)
	}
	// for _, params := range c.Lakeshores {
	// ctlm := lakeshore.New
	// }
	for _, params := range c.CryoCons {
		mon := cryocon.NewTemperatureMonitor(params.Addr, params.URLStem)
		mf.Add(mon)
	}

	for _, params := range c.NKTs {
		main := nkt.NewSuperKExtreme(params.Addr, params.URLStem, params.Serial)
		varia := nkt.NewSuperKVaria(params.Addr, params.URLStem, params.Serial)
		mf.Add(main)
		mf.Add(varia)
	}

	return mf
}
