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

	"github.jpl.nasa.gov/bdube/golab/fsm"
	"github.jpl.nasa.gov/bdube/golab/generichttp/motion"
	"github.jpl.nasa.gov/bdube/golab/pi"
)

func main() {
	args := os.Args[1:]
	// create the PI device and HTTP wrapper to it
	ctl := pi.NewController(args[0], false)
	dV := float64(25)
	ctl.DV = &dV
	wrap := motion.NewHTTPMotionController(ctl)

	dist := &fsm.Disturbance{
		PL:     pctl.PhaseLock{Interval: 40 * time.Millisecond},
		Repeat: false}

	// now create the disturbance engine and its HTTP wrapper
	cb := func(axes []string, pos []float64) {
		var err error
		if len(axes) == 1 {
			// err = ctl.SetVoltageSafe(axes[0], pos[0])
			err = ctl.SetVoltage(axes[0], pos[0])
		} else {
			err = ctl.MultiAxisMoveAbs(axes, pos)
		}
		if err != nil {
			log.Println(err)
		}
		if dist.Cursor%100 == 0 {
			err = ctl.PopError()
		}
	}
	dist.Callback = cb
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
