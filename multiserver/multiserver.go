package multiserver

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"github.jpl.nasa.gov/HCIT/go-hcit/generichttp"
	"github.jpl.nasa.gov/HCIT/go-hcit/server/middleware/locker"
	"github.jpl.nasa.gov/HCIT/go-hcit/thorlabs"

	"github.jpl.nasa.gov/HCIT/go-hcit/aerotech"
	"github.jpl.nasa.gov/HCIT/go-hcit/cryocon"
	"goji.io/pat"

	"github.jpl.nasa.gov/HCIT/go-hcit/nkt"

	"github.jpl.nasa.gov/HCIT/go-hcit/newport"

	"github.jpl.nasa.gov/HCIT/go-hcit/ixllightwave"
	"github.jpl.nasa.gov/HCIT/go-hcit/lesker"

	"github.jpl.nasa.gov/HCIT/go-hcit/commonpressure"
	"github.jpl.nasa.gov/HCIT/go-hcit/granvillephillips"

	"github.jpl.nasa.gov/HCIT/go-hcit/fluke"
	"github.jpl.nasa.gov/HCIT/go-hcit/server"

	"github.jpl.nasa.gov/HCIT/go-hcit/generichttp/ascii"

	"github.com/go-yaml/yaml"
	"goji.io"
)

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
}

// Config is a struct that holds the initialization parameters for various
// HTTP adapted devices.  It is to be populated by a json/unmarshal call.
type Config struct {
	// Addr is the address to listen at
	Addr string `yaml:"Addr"`

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
func BuildMux(c Config) *goji.Mux {
	// make the root handler
	root := goji.NewMux()
	supergraph := map[string][]string{}

	// for every node specified, build a submux
	for _, node := range c.Nodes {
		var httper server.HTTPer
		switch strings.ToLower(node.Type) {

		case "aerotech", "ensemble":
			ensemble := aerotech.NewEnsemble(node.Addr, node.Serial)
			httper = aerotech.NewHTTPWrapper(ensemble)

		case "cryocon":
			cryo := cryocon.NewTemperatureMonitor(node.Addr)
			httper = cryocon.NewHTTPWrapper(*cryo)

		case "fluke", "dewk":
			dewK := fluke.NewDewK(node.Addr, node.Serial)
			httper = fluke.NewHTTPWrapper(*dewK)

		case "convectron", "gpconvectron":
			cv := granvillephillips.NewSensor(node.Addr, node.Serial)
			httper = commonpressure.NewHTTPWrapper(*cv)

		case "lightwave", "ldc3916", "ixl":
			ldc := ixllightwave.NewLDC3916(node.Addr)
			httper = ixllightwave.NewHTTPWrapper(*ldc)

		// reserved for lakeshore

		case "lesker", "kjc":
			kjc := lesker.NewSensor(node.Addr, node.Serial)
			httper = commonpressure.NewHTTPWrapper(*kjc)

		case "esp", "esp300", "esp301":
			esp := newport.NewESP301(node.Addr, node.Serial)
			httper = newport.NewESP301HTTPWrapper(esp)

		case "xps":
			xps := newport.NewXPS(node.Addr)
			httper = newport.NewXPSHTTPWrapper(xps)

		case "nkt", "superk":
			skE := nkt.NewSuperKExtreme(node.Addr, node.Serial)
			skV := nkt.NewSuperKVaria(node.Addr, node.Serial)
			httper = nkt.NewHTTPWrapper(*skE, *skV)

		case "itc4000", "tl-laser-diode":
			itc, err := thorlabs.NewITC4000()
			if err != nil {
				log.Fatal(err)
			}
			httper = generichttp.NewHTTPLaserController(itc)
			ascii.InjectRawComm(httper, itc)

		default:
			continue // could be an empty entry in the list of nodes
		}

		// prepare the URL, "omc/nkt" => "/omc/nkt/*"
		hndlS := server.SubMuxSanitize(node.Endpoint)

		// add the endpoints to the graph
		supergraph[hndlS] = httper.RT().Endpoints()

		// add a lock interface for this node
		lock := locker.New()
		locker.Inject(httper, lock)

		// bind to the mux
		mux := goji.SubMux()
		httper.RT().Bind(mux)

		// add the lock middleware
		mux.Use(lock.Check)
		root.Handle(pat.New(hndlS), mux)
	}
	root.HandleFunc(pat.Get("/endpoints"), func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err := json.NewEncoder(w).Encode(supergraph)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	return root
}
