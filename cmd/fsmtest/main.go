package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/brandondube/pctl"

	"goji.io"
	"goji.io/pat"

	"github.jpl.nasa.gov/HCIT/go-hcit/fsm"
	"github.jpl.nasa.gov/HCIT/go-hcit/generichttp/motion"
	"github.jpl.nasa.gov/HCIT/go-hcit/pi"
)

func main() {
	args := os.Args[1:]
	// create the PI device and HTTP wrapper to it
	ctl := pi.NewController(args[0], false)
	wrap := motion.NewHTTPMotionController(ctl)

	// now create the disturbance engine and its HTTP wrapper
	cb := func(axes []string, pos []float64) {
		err := ctl.MultiAxisMoveAbs(axes, pos)
		if err != nil {
			log.Println(err)
		}
	}
	dist := &fsm.Disturbance{
		Callback: cb,
		PL:       pctl.PhaseLock{Interval: 40 * time.Millisecond},
		Repeat:   false}
	wrap2 := fsm.NewHTTPDisturbance(dist)
	// set up the HTTP bindings to the server
	mainMux := goji.NewMux()
	subMux := goji.SubMux()
	wrap.RT().Bind(subMux)
	wrap2.RT().Bind(subMux)
	mainMux.Handle(pat.New("/fsm/*"), subMux)
	mainMux.HandleFunc(pat.Get("/fsm/endpoints"), func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err := json.NewEncoder(w).Encode(wrap.RT())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	log.Fatal(http.ListenAndServe(":5001", mainMux))
}
