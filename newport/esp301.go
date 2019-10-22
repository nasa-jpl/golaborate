package newport

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.jpl.nasa.gov/HCIT/go-hcit/comm"
	"github.jpl.nasa.gov/HCIT/go-hcit/server"

	"github.com/tarm/serial"
)

const (
	// ESP301RemoteBufferSize is the number of ASCII characters that fit in the buffer on the ESP301.
	ESP301RemoteBufferSize = 80
)

var (
	// ErrBufferWouldOverflow is generated when the buffer on the ESP controller
	// would overflow if the message was transmitted
	ErrBufferWouldOverflow = errors.New("buffer too long, maximum command length is 80 characters")
)

// Command describes a command
type Command struct {
	Cmd         string `json:"cmd"`
	Alias       string `json:"alias"`
	Description string `json:"description"`
	UsesAxis    bool   `json:"usesAxis"`
	IsReadOnly  bool   `json:"isReadOnly"`
}

// JSONCommand is a primitive describing a command sent as JSON.
// CMD may either be a command (Command.Cmd) or an alias (Command.Alias)
// if Write is true, the data (F64) will be used.  If false, it will be ignored.
type JSONCommand struct {
	Axis  int     `json:"axis"`
	Cmd   string  `json:"cmd"`
	F64   float64 `json:"f64"`
	Write bool    `json:"write"`
}

// ErrCommandNotFound is generated when a command is unknown to the newport module
type ErrCommandNotFound struct {
	Cmd string
}

func (e ErrCommandNotFound) Error() string {
	return fmt.Sprintf("command %s not found", e.Cmd)
}

// ErrAliasNotFound is generated when an alias is unknown to the newport module
type ErrAliasNotFound struct {
	Alias string
}

func (e ErrAliasNotFound) Error() string {
	return fmt.Sprintf("alias %s not found", e.Alias)
}

var (
	commands = []Command{
		// Status functions
		{Cmd: "TE", Alias: "err-num", Description: "get error number", IsReadOnly: true},
		{Cmd: "TP", Alias: "get-position", Description: "get position", UsesAxis: true, IsReadOnly: true},
		{Cmd: "TS", Alias: "controller-status", Description: "get controller status", IsReadOnly: true},
		{Cmd: "TV", Alias: "get-velocity", Description: "get velocity", UsesAxis: true, IsReadOnly: true},
		{Cmd: "TX", Alias: "controller-activity", Description: "get controller activity", IsReadOnly: true},
		{Cmd: "VE", Alias: "controller-firmware", Description: "get controller firmware version", IsReadOnly: true},

		// Motion functions
		{Cmd: "AB", Alias: "abort-program", Description: "abort program", UsesAxis: true},
		{Cmd: "DH", Alias: "define-home", Description: "define home", UsesAxis: true},
		{Cmd: "MT", Alias: "move-hw-limit", Description: "move to hardware limit", UsesAxis: true},
		{Cmd: "MV", Alias: "move-indef", Description: "move indefinitely", UsesAxis: true},
		{Cmd: "OR", Alias: "origin-search", Description: "origin searching", UsesAxis: true},
		{Cmd: "PA", Alias: "move-abs", Description: "move absolute", UsesAxis: true},
		{Cmd: "PR", Alias: "move-rel", Description: "move relative", UsesAxis: true},
		{Cmd: "ST", Alias: "stop", Description: "stop motion", UsesAxis: true},

		// trajectory definition, less JH JK JW OL OH OM SH UF
		{Cmd: "AC", Alias: "set-accel", Description: "set acceleration", UsesAxis: true},
		{Cmd: "AE", Alias: "set-estop-accel", Description: "set e-stop acceleration", UsesAxis: true},
		{Cmd: "AG", Alias: "set-decel", Description: "set deceleration", UsesAxis: true},
		{Cmd: "AU", Alias: "set-max-accel", Description: "set maximum acceleration", UsesAxis: true},
		{Cmd: "BA", Alias: "set-backlash-comp", Description: "set backlash compensation on or off", UsesAxis: true},
		{Cmd: "CO", Alias: "set-linear-comp", Description: "set linear compensation on or off", UsesAxis: true},
		{Cmd: "VA", Alias: "set-velocity-linear", Description: "set velocity for linear motors", UsesAxis: true},
		{Cmd: "VB", Alias: "set-velocity-stepper", Description: "set velocity for stepper motors", UsesAxis: true},
		{Cmd: "VU", Alias: "set-max-speed", Description: "set maximum speed", UsesAxis: true},

		// Not implemented:
		// - General mode selection,
		// - motion device parameters,
		// - programming,
		// - flow control & sequence,
		// - group functions,
		// - digital filters
		// - master-slave mode definition
	}
)

func commandFromCmd(cmd string) (Command, error) {
	for _, c := range commands {
		if c.Cmd == cmd {
			return c, nil
		}
	}
	return Command{}, ErrCommandNotFound{cmd}
}

func commandFromAlias(alias string) (Command, error) {
	for _, c := range commands {
		if c.Alias == alias {
			return c, nil
		}
	}
	return Command{}, ErrAliasNotFound{alias}
}

func commandFromCmdOrAlias(cmdAlias string) (Command, error) {
	var cmd Command
	cmd, err := commandFromCmd(cmdAlias)
	if err != nil {
		cmd, err = commandFromAlias(cmdAlias)
	}
	return cmd, err
}

// makeSerConf makes a new serial.Config with correct parity, baud, etc, set.
func makeSerConf(addr string) *serial.Config {
	return &serial.Config{
		Name:        addr,
		Baud:        19200,
		Size:        8,
		Parity:      serial.ParityNone,
		StopBits:    serial.Stop1,
		ReadTimeout: 1 * time.Second}
}

// ESP301 represents an ESP301 motion controller
type ESP301 struct {
	*comm.RemoteDevice
	server.Server
}

// SerialConf returns a serial config and satisfies SerialConfigurator
func (esp *ESP301) SerialConf() serial.Config {
	return *makeSerConf(esp.Addr)

}

// NewESP301 makes a new ESP301 motion controller instance
func NewESP301(addr, urlStem string, serial bool) *ESP301 {
	rd := comm.NewRemoteDevice(addr, serial, nil, makeSerConf(addr))
	srv := server.NewServer(urlStem)
	esp := ESP301{RemoteDevice: &rd}
	srv.RouteTable["raw"] = esp.HTTPRaw
	srv.RouteTable["single-cmd"] = esp.HTTPJSONSingle
	srv.RouteTable["multi-cmd"] = esp.HTTPJSONArray
	srv.RouteTable["cmd-list"] = esp.HTTPCmdList
	srv.RouteTable["simple-pos-abs"] = esp.HTTPPosAbs
	esp.Server = srv
	return &esp
}

// RawCommand sends a command directly to the motion controller (with EOT appended) and returns the response as-is
func (esp *ESP301) RawCommand(cmd string) (string, error) {
	err := esp.Open()
	if err != nil {
		return "", err
	}
	defer esp.CloseEventually()
	r, err := esp.SendRecv([]byte(cmd))
	if err != nil {
		return "", err
	}
	return string(r), nil

}

// HTTPPosAbs gets the absolute position of an axis on GET or sets it on POST
func (esp *ESP301) HTTPPosAbs(w http.ResponseWriter, r *http.Request) {
	return
}

// HTTPRaw handles requests with raw string payloads
func (esp *ESP301) HTTPRaw(w http.ResponseWriter, r *http.Request) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fstr := fmt.Sprintf("unable to decode string from query.  Query must be a JSON request with \"str\" field. %q", err)
		log.Println(fstr)
		http.Error(w, fstr, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	resp, err := esp.RawCommand(string(b))
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	b = append([]byte(resp), '\n')
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Length", string(len(b)))
	w.Write(b)
}

// HTTPJSONSingle handles singular commands over HTTP of JSONCommand type
func (esp *ESP301) HTTPJSONSingle(w http.ResponseWriter, r *http.Request) {
	jcmd := &JSONCommand{}
	err := json.NewDecoder(r.Body).Decode(jcmd)
	defer r.Body.Close()
	if err != nil {
		fstr := fmt.Sprintf("error decoding JSON, request should have 3 fields; \"axis\", \"cmd\", \"f64\".  axis and f64 may be left blank.  %q", err)
		log.Println(err)
		http.Error(w, fstr, http.StatusBadRequest)
		return
	}
	cmd, err := commandFromCmdOrAlias(jcmd.Cmd)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tele := makeTelegram(cmd, jcmd.Axis, jcmd.Write, jcmd.F64)
	err = esp.Open()
	if err != nil {
		fstr := fmt.Sprintf("error opening connection to motion controller %q", err)
		log.Println(err)
		http.Error(w, fstr, http.StatusInternalServerError)
		return
	}
	defer esp.CloseEventually()
	resp, err := esp.SendRecv([]byte(tele))
	if err != nil {
		fstr := fmt.Sprintf("error communicating with motion controller %q.  Received response %q", err, resp)
		log.Println(err)
		http.Error(w, fstr, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Length", string(len(resp)))
	w.Write(append(resp, byte('\n')))
}

// HTTPJSONArray handles arrays of commands over HTTP of JSONCommand type
func (esp *ESP301) HTTPJSONArray(w http.ResponseWriter, r *http.Request) {
	jcmds := []JSONCommand{}
	err := json.NewDecoder(r.Body).Decode(&jcmds)
	defer r.Body.Close()
	if err != nil {
		fstr := fmt.Sprintf("error decoding JSON, request should have 3 fields; \"axis\", \"cmd\", \"f64\".  axis and f64 may be left blank.  %q", err)
		log.Println(err)
		http.Error(w, fstr, http.StatusBadRequest)
	}
	l := len(jcmds)
	cmds := make([]Command, 0, l)
	axes := make([]int, 0, l)
	writes := make([]bool, 0, l)
	datas := make([]float64, 0, l)

	for _, c := range jcmds {
		cmd, err := commandFromCmdOrAlias(c.Cmd)
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		cmds = append(cmds, cmd)
		axes = append(axes, c.Axis)
		writes = append(writes, c.Write)
		datas = append(datas, c.F64)
	}

	tele := makeTelegramPlural(cmds, axes, writes, datas)
	if l := len(tele); l > ESP301RemoteBufferSize {
		err = ErrBufferWouldOverflow
	}
	if err != nil {
		fstr := fmt.Sprintf("command sequence would result in buffer overflow on the motion controller, len>80 %s", tele)
		log.Println(err)
		http.Error(w, fstr, http.StatusInternalServerError)
		return
	}

	err = esp.Open()
	if err != nil {
		fstr := fmt.Sprintf("error opening connection to motion controller %q", err)
		log.Println(err)
		http.Error(w, fstr, http.StatusInternalServerError)
		return
	}
	defer esp.CloseEventually()
	resp, err := esp.SendRecv([]byte(tele))
	if err != nil {
		fstr := fmt.Sprintf("error communicating with motion controller %q", err)
		log.Println(err)
		http.Error(w, fstr, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Length", string(len(resp)))
	w.Write(append(resp, '\n'))
}

// HTTPCmdList returns a list of command objects which include:
// cmd (what Newport sees),
// alias (a friendier name you may use)
// description (a brief description)
// isReadOnly (whether the command is read-only or not)
// usesAxis (whether the command uses an axis or not)
func (esp *ESP301) HTTPCmdList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(commands)
	if err != nil {
		fstr := fmt.Sprintf("json encoding error %q", err)
		log.Println(fstr)
		http.Error(w, fstr, http.StatusInternalServerError)
		return
	}
	return
}

func makeTelegram(c Command, axis int, write bool, data float64) string {
	pieces := []string{}
	if c.UsesAxis {
		pieces = append(pieces, strconv.Itoa(axis))
	}
	pieces = append(pieces, c.Cmd)
	if c.IsReadOnly || !write {
		pieces = append(pieces, "?")
	} else {
		pieces = append(pieces, strconv.FormatFloat(data, 'g', -1, 64))
	}
	return strings.Join(pieces, "")
}

func makeTelegramPlural(c []Command, axes []int, write []bool, data []float64) string {
	telegrams := make([]string, 0, len(c))
	for idx, c := range c {
		telegrams = append(telegrams, makeTelegram(c, axes[idx], write[idx], data[idx]))
	}
	return strings.Join(telegrams, ";")
}
