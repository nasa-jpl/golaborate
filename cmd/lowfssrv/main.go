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
	"github.jpl.nasa.gov/bdube/golab/generichttp/camera"
	"github.jpl.nasa.gov/bdube/golab/server"
	"goji.io/pat"

	"github.com/brandondube/pctl"
)

const (
	// RESPOK is the response sent to the reconstruction client
	// if the command was accepted
	RESPOK = 'O'

	// RESPNOK is the response sent to the reconstruction client
	// if the command was not accepted
	RESPNOK = 'N'
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

	// CommIn is the channel used to command the loop
	CommIn chan string

	// CommOut is the channel used to feed back to the controller
	CommOut chan []byte

	// LastSourceSocket indicates if the command came from the outside
	// this is used to indicate if a reply should be sent
	LastSourceSocket bool

	// PL is the phase lock on the loop
	PL pctl.PhaseLock

	// aoi is the AOI used on the camera
	aoi camera.AOI

	// stride is the width of a row in the AOI, in bytes
	stride int
}

// Loop runs the loop, reading frames from the camera and
// passing replies to the FSM
func (l *LOWFS) Loop() {
	for {
		msg := <-l.CommIn // implicitly assume only stop comes from in or outside
		// would use switch, but want to partially compare
		// // short circuit blank shots
		// if msg == "" {
		// 	continue
		// }
		if msg == "frame?" {
			err := l.Cam.QueueBuffer()
			if err != nil {
				log.Println(err)
			}
			err = l.Cam.WaitBuffer(l.PL.Interval * 5)
			if err != nil {
				log.Println(err)
			}
			buf, err := l.Cam.Buffer()
			if err != nil {
				log.Println(err)
			}
			buf = sdk3.UnpadBuffer(buf, l.stride, l.aoi.Width, l.aoi.Height)
			l.CommOut <- buf
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
			// some kind of FSM command implementation
			// fsmchan <- floats
			l.CommOut <- []byte{RESPOK}
		} else if msg == "stop" {
			if l.LastSourceSocket {
				l.CommOut <- []byte{RESPOK}
			}
			return
		} else {
			if l.LastSourceSocket {
				l.CommOut <- []byte{RESPNOK}
			}
		}
	}
}

// HandleSocket spawns a pair of goroutines
// that handle read and writes from the socket
func (l *LOWFS) HandleSocket() {
	go func() {
		for {
			msg, err := l.Conn.Recv(0)
			if err != nil {
				log.Println(err)
			}
			l.CommIn <- msg
			l.LastSourceSocket = true
			l.Conn.SendBytes(<-l.CommOut, 0)
		}
	}()
}

// Start configures the AOI and begins the loop
func (l *LOWFS) Start(w http.ResponseWriter, r *http.Request) {
	aoi := camera.AOI{}
	err := json.NewDecoder(r.Body).Decode(&aoi)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = l.Cam.SetAOI(aoi)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	aoi, err = l.Cam.GetAOI()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	stride, err := l.Cam.GetAOIStride()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// swap width and height on the aoi to undo the SDK
	width := aoi.Width
	height := aoi.Height
	aoi.Width = height
	aoi.Height = width
	l.aoi = aoi
	l.stride = stride

	fps, err := sdk3.GetFloat(l.Cam.Handle, "FrameRate")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	interFrameTime := 1 / fps // seconds
	interFrameTimeDur := time.Duration(interFrameTime * 1e9)
	l.PL.Interval = interFrameTimeDur
	err = sdk3.SetEnumString(l.Cam.Handle, "CycleMode", "Continuous")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = sdk3.IssueCommand(l.Cam.Handle, "AcquisitionStart")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	go l.Loop()
	w.WriteHeader(http.StatusOK)
}

// Stop ceases operation of the loop on the camera
func (l *LOWFS) Stop(w http.ResponseWriter, r *http.Request) {
	err := sdk3.IssueCommand(l.Cam.Handle, "AcquisitionStop")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = sdk3.SetEnumString(l.Cam.Handle, "CycleMode", "Fixed")
	l.CommIn <- "stop"
	l.LastSourceSocket = false
}

func openCamera() (*sdk3.Camera, error) {
	// now, the andor camera
	err := sdk3.InitializeLibrary()
	if err != nil {
		log.Fatal(err)
	}
	ncam, err := sdk3.DeviceCount()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("There are %d cameras connected\n", ncam)
	swver, err := sdk3.SoftwareVersion()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("SDK version is %s\n", swver)

	// now scan for the right serial number
	// c escapes into the outer scope
	sn := "auto"
	var (
		c     *sdk3.Camera
		snCam string
	)
	for idx := 0; idx < ncam; idx++ {
		c, err = sdk3.Open(idx)
		if err != nil {
			log.Fatal(err)
		}
		snCam, err = c.GetSerialNumber()
		if err != nil {
			c.Close()
			log.Fatal(err)
		}
		if !strings.Contains(sn, "SFT") {
			break
		} else {
			c.Close()
		}
	}
	model, err := c.GetModel()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("connected to %s SN %s\n", model, snCam)

	// preamp gains relevant to lowfs
	// 12-bit (high well capacity)
	// 12-bit (low noise)
	cfg := map[string]interface{}{
		"ElectronicShutteringMode": "Rolling",
		"SimplePreAmpGainControl":  "16-bit (low noise & high well capacity)",
		"FanSpeed":                 "Off",
		"PixelReadoutRate":         "280 MHz",
		"PixelEncoding":            "Mono16",
		"TriggerMode":              "Internal",
		"MetadataEnable":           false,
		"SensorCooling":            false,
		"SpuriousNoiseFilter":      false,
		"StaticBlemishCorrection":  false}
	err = c.Configure(cfg)
	if err != nil {
		log.Fatal(err)
	}
	c.Allocate()
	err = c.QueueBuffer()
	return c, err
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

	// now set up the LOWFS communication
	// first, ZMQ
	ctx, err := zmq4.NewContext()
	if err != nil {
		log.Fatal(err)
	}
	socket, err := ctx.NewSocket(zmq4.REP)
	if err != nil {
		log.Fatal(err)
	}
	defer socket.Close()
	// err = socket.Bind("ipc:///tmp/lowfszmq")
	err = socket.Bind("tcp://*:7999")
	if err != nil {
		log.Fatal(err)
	}

	// last, the camera and HTTP interface
	c, err := openCamera()
	if err != nil {
		log.Fatal(err)
	}
	w := sdk3.NewHTTPWrapper(c, nil)
	lowfs := LOWFS{Conn: socket, Cam: c, CommIn: make(chan string), CommOut: make(chan []byte), PL: pl}
	lowfs.HandleSocket()

	root := goji.NewMux()
	mux := goji.SubMux()
	w.RT()[pat.Post("/start-continuous-loop")] = lowfs.Start
	w.RT()[pat.Post("/stop-continuous-loop")] = lowfs.Stop
	rt[pat.Get("/interval")] = getInterval
	rt[pat.Post("/interval")] = setInterval

	rt.Bind(root)
	w.RT().Bind(mux)
	root.Handle(pat.New("/camera/*"), mux)
	http.ListenAndServe(":8000", root)
}
