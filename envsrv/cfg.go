package envsrv

import (
	"os"

	"goji.io"
	"goji.io/pat"

	"gopkg.in/yaml.v2"
)

// ObjSetup holds the typical triplet of args for a New<device> call.
// Serial is not always used, and need not be populated in the config file
// if not used.
type ObjSetup struct {
	// Addr holds the network or filesystem address of the remote device,
	// e.g. 192.168.100.123:2006 for a device connected to port 6
	// on a digi portserver
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

	// Network is a flat listing of network branches.  It is recursed over to produce a tree of Muxes with goji
	Network []Node
}

// BuildNetwork returns a Goji mux that is populated with submuxes for the network
// branches
func BuildNetwork(nodes []Node) *goji.Mux {
	// make the root mux
	root := goji.NewMux()

	// make a channel for stuff that has to processed because the parent is missing
	// and close it on completion
	reprocess := make(chan Node, len(nodes))
	defer close(reprocess)

	// then make a map of muxes so we know when the parents are made
	muxes := make(map[string]*goji.Mux)

	for _, node := range nodes {
		// first pass, anything with a Parent has to be reprocessed
		if node.Parent != "" {
			reprocess <- node
			continue
		}
		// otherwise, make a submux and put it on root
		ptrn := pat.New("/" + node.Name + "/*")
		mux := goji.SubMux()
		muxes[node.Parent+node.Name] = mux
		root.Handle(ptrn, mux)
	}

	for node := range reprocess {
		// if the parent exists, make our mux and bind to it
		if parent, ok := muxes[node.Parent+node.Name]; ok {
			mux := goji.SubMux()
			parent.Handle(pat.New("/"+node.Name+"/*"), mux)
			muxes[node.Parent+node.Name] = mux
		} else {
			reprocess <- node
		}
	}

	return root
}

// Node is a piece of a network tree.
type Node struct {
	// Parent is the parent node, if there is one
	Parent string `yaml:"parent"`

	// Name is the name of this node, if there is one
	Name string `yaml:"name"`

	// Children are the nodes connected below this one
	Children []Node
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
