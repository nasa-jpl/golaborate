package main

import (
	"encoding/json"
	"go/types"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pebbe/zmq4"

	"goji.io"

	"github.jpl.nasa.gov/bdube/golab/andor/sdk3"
	"github.jpl.nasa.gov/bdube/golab/server"
	"goji.io/pat"

	"github.com/brandondube/pctl"
)

// LOWFS is a type that manages the camera generating data and
// the replies from the reconstructor
type LOWFS struct {
	// Conn is the connection.  One way to the reconstructor
	// sends blobs of image data (just the array buffer, load with np.frombuffer)
	// and receives ASCII/CSV encoded FSM commands
	Conn *zmq4.Socket

	// Cam holds the camera, which can be managed and generates the feedback
	// to drive the FSM loop
	Cam *sdk3.Camera
}

// Loop runs the loop, reading frames from the camera and
// passing replies to the FSM
func (l *LOWFS) Loop(fsmchan chan<- []float64) {
	socket := l.Conn
	for {
		// this loop waits for data from the reconstructor
		// which does not live in this program
		msg, err := socket.Recv(0)
		if err != nil {
			log.Println(err)
		}
		// would use switch, but want to partially compare
		if msg == "frame?" {
			// get frame from camera
		} else if msg[:3] == "fsm" {
			// msg is CSV floats to send to the control loop
			// split off the front
			msg = msg[3:]
			// chunk by "," and parse floats
			chunks := strings.Split(msg, ",")
			floats := make([]float64, len(chunks))
			for i := 0; i < 3; i++ {
				f, err := strconv.ParseFloat(chunks[i], 64)
				if err != nil {
					log.Println(err)
				}
				floats[i] = f
			}
			fsmchan <- floats
			socket.SendBytes([]byte{6}, 0) // 6 == ACK
		}
		else {
			socket.SendBytes([]byte{21}, 0) // 21 == NACK
		}
	}
}

func main() {
	// create the table of routes used to administrate this control system,
	// which will be populated as we initialize the pieces
	rt := server.RouteTable{}

	// create the phase lock used to make sure we run at the specified period
	// and bind its meta-routes to the table
	pl := pctl.PhaseLock{Interval: 2 * time.Millisecond}
	setInterval := func(w http.ResponseWriter, r *http.Request) {
		str := server.StrT{}
		err := json.NewDecoder(r.Body).Decode(&str)
		defer r.Body.Close()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		dur, err := time.ParseDuration(str.Str)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		pl.Interval = dur
		w.WriteHeader(http.StatusOK)
		return
	}
	getInterval := func(w http.ResponseWriter, r *http.Request) {
		hp := server.HumanPayload{T: types.String, String: pl.Interval.String()}
		hp.EncodeAndRespond(w, r)
		return
	}
	rt[pat.Get("/interval")] = getInterval
	rt[pat.Post("/interval")] = setInterval
	// cl := fsm.ControlLoop{}

	// now set up the LOWFS communication
	ctx, err := zmq4.NewContext()
	if err != nil {
		log.Fatal(err)
	}
	socket, err := ctx.NewSocket(zmq4.REP)
	if err != nil {
		log.Fatal(err)
	}
	defer socket.Close()
	err = socket.Bind("tcp://*:8001")
	if err != nil {
		log.Fatal(err)
	}
	lowfs := LOWFS{Conn: socket, Cam: nil}

	mux := goji.NewMux()
	rt.Bind(mux)
	go http.ListenAndServe(":8000", mux)
}
