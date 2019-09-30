package sensor

// DataFunc returns << something >> from an address
type DataFunc func(addr string) (interface{}, error)

// Info holds information about a sensor
type Info struct {
	Addr     string `yaml:"addr"`
	Name     string `yaml:"name"`
	Conntype string `yaml:"conntype"`
	Type     string `yaml:"type"`
	Func     DataFunc
}
