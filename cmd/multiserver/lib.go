package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/nasa-jpl/golaborate/agilent"
	"github.com/nasa-jpl/golaborate/generichttp"
	"github.com/nasa-jpl/golaborate/keysight"
	"github.com/nasa-jpl/golaborate/pi"
	"github.com/nasa-jpl/golaborate/server/middleware/locker"
	"github.com/nasa-jpl/golaborate/util"

	"github.com/nasa-jpl/golaborate/aerotech"
	"github.com/nasa-jpl/golaborate/cryocon"

	"github.com/nasa-jpl/golaborate/nkt"

	"github.com/nasa-jpl/golaborate/newport"

	"github.com/nasa-jpl/golaborate/fluke"

	"github.com/nasa-jpl/golaborate/generichttp/ascii"
	"github.com/nasa-jpl/golaborate/generichttp/motion"
	"github.com/nasa-jpl/golaborate/generichttp/tmc"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-yaml/yaml"
)

// Minmax holds a min and max value
type Minmax struct {
	Min float64 `yaml:"Min"`
	Max float64 `yaml:"Max"`
}

// Daisy holds a controller ID, endpoint, and limit
type Daisy struct {
	ControllerID int               `yaml:"ControllerID"`
	Endpoint     string            `yaml:"Endpoint"`
	Limits       map[string]Minmax `yaml:"Limits"`
}

// ObjSetup holds the typical triplet of args for a New<device> call.
// Serial is not always used, and need not be populated in the config file
// if not used.
type ObjSetup struct {
	// Addr holds the network or filesystem address of the remote device,
	// e.g. 192.168.100.123:2006 for a device connected to port 6
	// on a digi portserver, or /dev/ttyS4 for an RS232 device on a serial cable
	Addr string `yaml:"Addr"`

	// URL is the full path the routes from this device will be served on
	// ex. URL="/omc/nkt" will produce routes of /omc/nkt/power, etc.
	Endpoint string `yaml:"Endpoint"`

	// Endpt is the final "directory" to put object functionality under, it will be
	// prepended to routes
	// Serial determines if the connection is serial/RS232 (True) or TCP (False)
	Serial bool `yaml:"Serial"`

	// Typ is the "type" of the object, e.g. ESP301
	Type string `yaml:"Type"`

	// Args holds any arguments to pass into the constructor for the object
	Args map[string]interface{} `yaml:"Args"`

	DaisyChain []Daisy `yaml:"DaisyChain"`
}

// Config is a struct that holds the initialization parameters for various
// HTTP adapted devices.  It is to be populated by a json/unmarshal call.
type Config struct {
	// Addr is the address to listen at
	Addr string `yaml:"Addr"`

	Mock bool `yaml:"Mock"`

	// Nodes is the list of nodes to set up
	Nodes []ObjSetup `yaml:"Nodes"`
}

// LoadYaml converts a (path to a) yaml file into a Config struct
func LoadYaml(path string) (Config, error) {
	cfg := Config{}
	f, err := os.Open(path)
	if err != nil {
		return cfg, err
	}

	err = yaml.NewDecoder(f).Decode(&cfg)
	return cfg, err
}

// BuildMux takes equal length slices of HTTPers and strings ("stems")
// and uses them to construct a goji mux with populated handlers.
// The mux serves a special route, route-list, which returns an
// array of strings containing all routes as JSON.
func BuildMux(c Config) chi.Router {
	// make the root handler
	root := chi.NewRouter()
	root.Use(middleware.Logger)
	supergraph := map[string][]string{}

OuterLoop:
	// for every node specified, build a submux
	for _, node := range c.Nodes {
		var (
			httper     generichttp.HTTPer
			middleware []func(http.Handler) http.Handler
		)
		axislocker := false
		typ := strings.ToLower(node.Type)
		switch typ {

		case "aerotech", "ensemble", "esp", "esp300", "esp301", "picomotor", "xps", "pi", "pi-daisy-chain":
			axislocker = true
			/* the limits are encoded as:
			Args:
				Limits:
					X:
						Min: 0
						Max: 1
					Y:
						...

			So, this translates to Go:
			Args -> map[string]interface
			Limits -> map[string]interface
			limit key -> map[string]float64
			*/
			limiters := map[string]util.Limiter{}
			if node.Args != nil {
				if node.Args["Limits"] != nil {
					rawlimits := node.Args["Limits"].(map[string]interface{})
					for k, v := range rawlimits {
						limiter := util.Limiter{}
						if min, ok := v.(map[string]interface{})["Min"]; ok {
							limiter.Min = min.(float64)
						}
						if max, ok := v.(map[string]interface{})["Max"]; ok {
							limiter.Max = max.(float64)
						}
						limiters[k] = limiter
					}
				}
			}
			switch typ {
			case "aerotech", "ensemble":
				if c.Mock {
					log.Fatal("Aerotech mock interface is not yet implemented")
				}
				ensemble := aerotech.NewEnsemble(node.Addr, node.Serial)
				limiter := motion.LimitMiddleware{Limits: limiters, Mov: ensemble}
				httper = motion.NewHTTPMotionController(ensemble)
				middleware = append(middleware, limiter.Check)
				limiter.Inject(httper)
			case "esp", "esp300", "esp301":
				if c.Mock {
					log.Fatal("newport esp mock interface is not yet implemented")
				}
				esp := newport.NewESP301(node.Addr, node.Serial)
				limiter := motion.LimitMiddleware{Limits: limiters, Mov: esp}
				httper = motion.NewHTTPMotionController(esp)
				middleware = append(middleware, limiter.Check)
				limiter.Inject(httper)
			case "picomotor":
				pico := newport.NewPicomotor(node.Addr, node.Serial)
				limiter := motion.LimitMiddleware{Limits: limiters, Mov: pico}
				httper = motion.NewHTTPMotionController(pico)
				middleware = append(middleware, limiter.Check)
				limiter.Inject(httper)
			case "xps":
				var xps motion.Controller
				if c.Mock {
					xps = newport.NewControllerMock(node.Addr)
				} else {
					xps = newport.NewXPS(node.Addr)
				}
				limiter := motion.LimitMiddleware{Limits: limiters, Mov: xps}
				httper = motion.NewHTTPMotionController(xps)
				middleware = append(middleware, limiter.Check)
				limiter.Inject(httper)
			case "pi-daisy-chain":
				// daisy chain is special in that a single pool is used for multiple controllers
				network := pi.NewNetwork(node.Addr, node.Serial)
				for i := range node.DaisyChain {
					daisy := node.DaisyChain[i]
					ctl := network.Add(daisy.ControllerID, true, c.Mock) // true => handshaking//error checking
					limiter := motion.LimitMiddleware{Limits: limiters, Mov: ctl}
					httper = motion.NewHTTPMotionController(ctl)
					ascii.InjectRawComm(httper.RT(), ctl)
					limiter.Inject(httper)
					middleware = append(middleware, limiter.Check)
					// prepare the URL, "omc/nkt" => "/omc/nkt/*"
					hndlS := generichttp.SubMuxSanitize(daisy.Endpoint)

					// add a lock interface for this node
					var lock locker.ManipulableLock
					if !axislocker {
						lock = locker.New()
					} else {
						lock = locker.NewAL()
					}
					// add the lock middleware
					locker.Inject(httper, lock)
					r := chi.NewRouter()
					r.Use(middleware...)
					r.Use(lock.Check)
					httper.RT().Bind(r)
					root.Mount(hndlS, r)
				}
				continue OuterLoop
			case "pi":
				network := pi.NewNetwork(node.Addr, node.Serial)
				ctl := network.Add(1, true, c.Mock)
				limiter := motion.LimitMiddleware{Limits: limiters, Mov: ctl}
				httper = motion.NewHTTPMotionController(ctl)
				ascii.InjectRawComm(httper.RT(), ctl)
				limiter.Inject(httper)
				middleware = append(middleware, limiter.Check)

			}

		case "cryocon":
			if c.Mock {
				log.Fatal("cryocon mock interface is not yet implemented")
			}
			cryo := cryocon.NewTemperatureMonitor(node.Addr)
			httper = cryocon.NewHTTPWrapper(*cryo)

		case "fluke", "dewk":
			if c.Mock {
				log.Fatal("fluke dewk mock interface is not yet implemented")
			}
			dewK := fluke.NewDewK(node.Addr)
			httper = fluke.NewHTTPWrapper(*dewK)

		case "keysight-scope":
			if c.Mock {
				log.Fatal("keysight scope mock interface is not yet implemented")
			}
			scope := keysight.NewScope(node.Addr)
			httper = tmc.NewHTTPOscilloscope(scope)

		case "agilent-function-generator":
			if c.Mock {
				log.Fatal("agilent function generator mock interface is not yet implemented")
			}
			gen := agilent.NewFunctionGenerator(node.Addr, node.Serial)
			httper = tmc.NewHTTPFunctionGenerator(gen)

		case "keysight-daq":
			if c.Mock {
				log.Fatal("keysight daq xps mock interface is not yet implemented")
			}
			daq := keysight.NewDAQ(node.Addr)
			httper = tmc.NewHTTPDAQ(daq)

		case "nkt", "superk":
			var sk nkt.AugmentedLaserController

			if c.Mock {
				sk = nkt.NewMockSuperK(node.Addr, node.Serial)
			} else {
				sk = nkt.NewSuperK(node.Addr, node.Serial)
			}
			httper = nkt.NewHTTPWrapper(sk)

		default:
			log.Fatal("type ", typ, " not understood")
		}

		// prepare the URL, "omc/nkt" => "/omc/nkt/*"
		hndlS := generichttp.SubMuxSanitize(node.Endpoint)

		// add the endpoints to the graph
		supergraph[hndlS] = httper.RT().Endpoints()

		// add a lock interface for this node
		var lock locker.ManipulableLock
		if !axislocker {
			lock = locker.New()
		} else {
			lock = locker.NewAL()
		}

		// add the lock middleware
		locker.Inject(httper, lock)

		// bind to the mux
		r := chi.NewRouter()
		r.Use(middleware...)
		r.Use(lock.Check)
		httper.RT().Bind(r)
		root.Mount(hndlS, r)
	}
	root.Get("/endpoints", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err := json.NewEncoder(w).Encode(supergraph)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	return root
}
