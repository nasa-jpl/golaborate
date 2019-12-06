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
