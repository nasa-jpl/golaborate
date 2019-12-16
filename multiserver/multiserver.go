package envsrv

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

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
	Addr string `yaml:"addr"`

	// URL is the full path the routes from this device will be served on
	// ex. URL="/omc/nkt" will produce routes of /omc/nkt/power, etc.
	URL string `yaml:"endpoint"`

	// Endpt is the final "directory" to put object functionality under, it will be
	// prepended to routes
	// Serial determines if the connection is serial/RS232 (True) or TCP (False)
	Serial bool `yaml:"serial"`
}

// Config is a struct that holds the initialization parameters for various
// HTTP adapted devices.  It is to be populated by a json/unmarshal call.
type Config struct {
	// Addr is the address to listen at
	Addr string

	// Flukes is a list of setup parameters that will automap to Fluke DewK objects
	Flukes []ObjSetup

	// GPConvectrons is a list of setup parameters that will automap to GP375 convectrons
	GPConvectrons []ObjSetup

	// Leskers is a list of setup parameters that will automap to KJC300s
	Leskers []ObjSetup

	// IXLLightwaves is a list of setup parameters that will automap to LDC3916 diode controllers
	IXLLightwaves []ObjSetup

	// Lakeshores is a list of setup parameters that will automap to Lakeshore 322 temperature controllers
	Lakeshores []ObjSetup

	// ESP300s is a list of setup parameters that will automap to ESP301 motion controllers
	ESP300s []ObjSetup

	// XPSs is a list of setup parameters that will automap to XPS motion controllers
	XPSs []ObjSetup

	// NKTs is a list of setup parameters that will automap to NKT supercontinuum lasers
	NKTs []ObjSetup

	// CryoCons is a list of setup parameters that will automap to crycon temperature monitors
	CryoCons []ObjSetup

	// Aerotechs is a list of setup parameters that will automap to Aerotech Ensemble controllers
	Aerotechs []ObjSetup
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
func (c Config) BuildMux() *goji.Mux {
	root := goji.NewMux()
	stems := []string{}
	httpers := []server.HTTPer{}

	for _, setup := range c.Flukes {
		dewK := fluke.NewDewK(setup.Addr, setup.Serial)
		httper := fluke.NewHTTPWrapper(*dewK)
		stems = append(stems, setup.URL)
		httpers = append(httpers, httper)
	}

	for _, setup := range c.GPConvectrons {
		cv := granvillephillips.NewSensor(setup.Addr, setup.Serial)
		httper := commonpressure.NewHTTPWrapper(*cv)
		stems = append(stems, setup.URL)
		httpers = append(httpers, httper)
	}

	for _, setup := range c.Leskers {
		kjc := lesker.NewSensor(setup.Addr, setup.Serial)
		httper := commonpressure.NewHTTPWrapper(*kjc)
		stems = append(stems, setup.URL)
		httpers = append(httpers, httper)
	}

	for _, setup := range c.IXLLightwaves {
		ldc := ixllightwave.NewLDC3916(setup.Addr)
		httper := ixllightwave.NewHTTPWrapper(*ldc)
		stems = append(stems, setup.URL)
		httpers = append(httpers, httper)
	}
	// for _, setup := range c.Lakeshores {
	// ctl := lakeshore.NewController()
	// }
	for _, setup := range c.ESP300s {
		esp := newport.NewESP301(setup.Addr, setup.Serial)
		httper := newport.NewESP301HTTPWrapper(esp)
		stems = append(stems, setup.URL)
		httpers = append(httpers, httper)
	}

	for _, setup := range c.NKTs {
		skE := nkt.NewSuperKExtreme(setup.Addr, setup.Serial)
		skV := nkt.NewSuperKVaria(setup.Addr, setup.Serial)
		httper := nkt.NewHTTPWrapper(*skE, *skV)
		stems = append(stems, setup.URL)
		httpers = append(httpers, httper)
	}

	for _, setup := range c.CryoCons {
		cryo := cryocon.NewTemperatureMonitor(setup.Addr)
		httper := cryocon.NewHTTPWrapper(*cryo)
		stems = append(stems, setup.URL)
		httpers = append(httpers, httper)
	}

	for _, setup := range c.Aerotechs {
		ensemble := aerotech.NewEnsemble(setup.Addr, setup.Serial)
		httper := aerotech.NewHTTPWrapper(*ensemble)
		stems = append(stems, setup.URL)
		httpers = append(httpers, httper)
	}

	supergraph := map[string][]string{}

	// the above just collected everything from the configs
	for idx := 0; idx < len(stems); idx++ {
		stem := stems[idx]
		httper := httpers[idx]
		mux := goji.SubMux()
		if !strings.HasPrefix(stem, "/") {
			stem = "/" + stem
		}
		if !strings.HasSuffix(stem, "/") {
			stem = stem + "/"
		}
		supergraph[stem] = httper.RT().Endpoints()
		stem = stem + "*"
		strP := pat.New(stem)
		root.Handle(strP, mux)
		httper.RT().Bind(mux)
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
