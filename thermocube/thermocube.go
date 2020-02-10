// Package thermocube enables working with 200~400 series temperature controllers.
package thermocube

import (
	"encoding/binary"
	"encoding/json"
	"math"
	"net"
	"net/http"
	"time"

	"github.com/tarm/serial"
	"github.jpl.nasa.gov/HCIT/go-hcit/comm"
	"github.jpl.nasa.gov/HCIT/go-hcit/generichttp/thermal"
	"github.jpl.nasa.gov/HCIT/go-hcit/server"
	"github.jpl.nasa.gov/HCIT/go-hcit/temperature"
	"github.jpl.nasa.gov/HCIT/go-hcit/util"
	"goji.io/pat"
)

// Direction describes the flow of data
type Direction bool

// RemoteControl describes if a feature is remote controlled or not
type RemoteControl bool

// Parameter holds five bits that map to unique parameters
type Parameter [5]bool

const (
	// HostToChiller is the direction of data flow where a message is sent to the chiller cube
	HostToChiller = Direction(true)

	// ChillerToHost is the direction of data flow where a message is sent to the host from the chiller cube
	ChillerToHost = Direction(false)

	// RemoteOn turns the chiller on
	RemoteOn = RemoteControl(true)

	// RemoteOff turns the chiller off
	RemoteOff = RemoteControl(false)
)

var (
	// ParamSetPoint is the temperature setpoint
	ParamSetPoint = Parameter{false, false, false, false, true}

	// ParamFluidTemp is the fluid temperature at the chiller output
	ParamFluidTemp = Parameter{false, true, false, false, true}

	// ParamFaults is the fault codes
	ParamFaults = Parameter{false, true, false, false, false}
)

// FaultState describes the faults of the cube
type FaultState struct {
	TankLevelLow bool `json:"tankLevelLow"`
	FanFail      bool `json:"fanFail"`
	PumpFail     bool `json:"pumpFail"`
	RTDOpen      bool `json:"rtdOpen"`
	RTDShort     bool `json:"rtdShort"`
}

// DecodeFault parses a fault from the controller
func DecodeFault(fault byte) FaultState {
	return FaultState{}
}

// DecodeTemp decodes the temperature and returns it as Celcius
func DecodeTemp(data []byte) temperature.Celsius {
	u := binary.LittleEndian.Uint16(data)
	f := float64(u) / 10
	F := temperature.Fahrenheit(f)
	return temperature.F2C(F)
}

// EncodeTemp encodes a temperature how the thermocube wants it
func EncodeTemp(t temperature.Celsius) []byte {
	f := temperature.C2F(t)
	i := uint16(math.Round(float64(f) * 10))
	buf := make([]byte, 2)
	binary.LittleEndian.PutUint16(buf, i)
	return buf
}

// Datagram is a payload to/from the thermocube
type Datagram struct {
	// Remote sets remote control On/Off
	Remote RemoteControl

	// On describes if the control system shall be enabled or disabled
	On RemoteControl

	// Dir is the communication direction
	Dir Direction

	// Param is what is being communicated
	Param Parameter

	// Data is the meat of the message
	Data []byte
}

// Encode converts the datagram into bytes
func (d Datagram) Encode() []byte {
	// see manual pg. 23
	// remotecontrol -> bit 7 (MSB)
	// on/off -> bit 6
	// direction -> bit 5
	// parameter -> 4~0
	// data -> next 1 or 2 bytes

	cmd := byte(0)
	cmd = util.SetBit(cmd, 7, bool(d.Remote))
	cmd = util.SetBit(cmd, 6, bool(d.On))
	cmd = util.SetBit(cmd, 5, bool(d.Dir))
	for idx := 4; idx > -1; idx-- {
		cmd = util.SetBit(cmd, uint(idx), d.Param[4-idx])
	}
	ret := []byte{cmd}
	if d.Data != nil && len(d.Data) != 0 {
		ret = append(ret, d.Data...)
	}
	return ret
}

// makeSerConf makes a new serial.Config with correct parity, baud, etc, set.
func makeSerConf(addr string) *serial.Config {
	return &serial.Config{
		Name:        addr,
		Baud:        9600,
		Size:        8,
		Parity:      serial.ParityNone,
		StopBits:    serial.Stop1,
		ReadTimeout: 1 * time.Second}
}

// Chiller describes a series 200~400 SolidState ThermoCube chiller
type Chiller struct {
	*comm.RemoteDevice
}

// NewChiller returns a new Chiller instance
func NewChiller(addr string, serial bool) *Chiller {
	// NewESP301 makes a new ESP301 motion controller instance
	rd := comm.NewRemoteDevice(addr, serial, nil, makeSerConf(addr))
	rd.Timeout = 3 * time.Second
	return &Chiller{RemoteDevice: &rd}
}

// Write sends a datagram to the controller.  The direction should be HostToChiller.
func (c *Chiller) Write(d Datagram) error {
	bytes := d.Encode()
	c.Lock()
	defer c.Unlock()
	c.LastComm = time.Now()
	conn, ok := (c.RemoteDevice.Conn).(net.Conn)
	if ok {
		conn.SetDeadline(time.Now().Add(c.Timeout))
	}
	_, err := c.RemoteDevice.Conn.Write(bytes)
	return err
}

// Read reads a value from a datagram.  The direction should be ChillerToHost.
func (c *Chiller) Read(d Datagram) ([]byte, error) {
	var nbytes int
	switch d.Param {
	case ParamFaults:
		nbytes = 1
	case ParamFluidTemp, ParamSetPoint:
		nbytes = 2
	}
	buf := make([]byte, nbytes)
	c.Lock()
	defer c.Unlock()
	// send the message to the cube
	gram := d.Encode()
	conn, ok := (c.RemoteDevice.Conn).(net.Conn)
	if ok {
		conn.SetDeadline(time.Now().Add(c.Timeout))
	}
	_, err := c.RemoteDevice.Conn.Write(gram)
	if err != nil {
		return buf, err
	}
	recieved := 0
	for recieved < nbytes {
		n, err := c.RemoteDevice.Conn.Read(buf)
		recieved += n
		if err != nil {
			return buf, err
		}
	}
	c.LastComm = time.Now()
	return buf, nil
}

// GetTemperatureSetpoint gets the current setpoint of the thermocube in Celcius
func (c *Chiller) GetTemperatureSetpoint() (float64, error) {
	d := Datagram{
		Remote: RemoteOn,
		On:     RemoteOn,
		Dir:    ChillerToHost,
		Param:  ParamSetPoint}
	err := c.Open()
	defer c.CloseEventually()
	if err != nil {
		return 0, err
	}
	resp, err := c.Read(d)
	if err != nil {
		return 0, err
	}
	return float64(DecodeTemp(resp)), nil
}

// SetTemperatureSetpoint sets the current setpoint of the thermocube in celcius
func (c *Chiller) SetTemperatureSetpoint(t float64) error {
	d := Datagram{
		Remote: RemoteOn,
		On:     RemoteOn,
		Dir:    HostToChiller,
		Param:  ParamSetPoint,
		Data:   EncodeTemp(temperature.Celsius(t))}

	err := c.Open()
	if err != nil {
		return err
	}
	defer c.CloseEventually()
	return c.Write(d)
}

// GetTemperature returns the current temperature at the chiller output in Celcius
func (c *Chiller) GetTemperature() (float64, error) {
	d := Datagram{
		Remote: RemoteOn,
		On:     RemoteOn,
		Dir:    ChillerToHost,
		Param:  ParamFluidTemp}

	err := c.Open()
	if err != nil {
		return 0, err
	}
	defer c.CloseEventually()
	resp, err := c.Read(d)
	if err != nil {
		return 0, err
	}
	return float64(DecodeTemp(resp)), nil
}

// GetFaults returns the faults of the controller
func (c *Chiller) GetFaults() (FaultState, error) {
	d := Datagram{
		Remote: RemoteOn,
		On:     RemoteOn,
		Dir:    ChillerToHost,
		Param:  ParamFaults}

	err := c.Open()
	defer c.CloseEventually()
	if err != nil {
		return FaultState{}, err
	}
	resp, err := c.Read(d)
	if err != nil {
		return FaultState{}, err
	}
	return DecodeFault(resp[0]), nil
}

// GetTankLevelLow returns true if the tank level is low (needs refilling)
func (c *Chiller) GetTankLevelLow() (bool, error) {
	fs, err := c.GetFaults()
	return fs.TankLevelLow, err
}

// RemoteStop toggles remote operation off
func (c *Chiller) RemoteStop() error {
	_, err := c.RemoteDevice.Conn.Write([]byte{0xA0})
	return err
}

// HTTPChiller is an HTTP wrapper around the thermocube
type HTTPChiller struct {
	server.RouteTable

	c *Chiller
}

// NewHTTPChiller returns a new HTTP wrapper around the chiller
func NewHTTPChiller(c *Chiller) HTTPChiller {
	rt := server.RouteTable{}
	thermal.HTTPController(c, rt)
	h := HTTPChiller{RouteTable: rt, c: c}
	rt[pat.Get("/faults")] = h.Faults
	return h
}

// RT satisfies server.HTTPer
func (h HTTPChiller) RT() server.RouteTable {
	return h.RouteTable
}

// Faults pipes the faults back over HTTP
func (h HTTPChiller) Faults(w http.ResponseWriter, r *http.Request) {
	f, err := h.c.GetFaults()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(f)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
