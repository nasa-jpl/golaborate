// Package ixllightwave contains code for operating IXL Lightwave LDC3916 laser diode controllers.
// It contains several single-value structs that are used to enable a "better"
// http interface where the return types are concrete and not strings, but
// they are burried behind a JSON field.  Each of these structs implements
// EncodeAndRespond, and the bodies of these functions are nearly copy pasted
// and can be ignored by the reader.
package ixllightwave

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.jpl.nasa.gov/HCIT/go-hcit/util"

	"github.jpl.nasa.gov/HCIT/go-hcit/comm"
	"github.jpl.nasa.gov/HCIT/go-hcit/server"
)

// the controller terminates with <CR> <NL> <END>
// it expects terminations of <NL> or <END> or <NL><END>
// we will use NL

const (
	// termination is the message termination used by the device
	termination = '\n'
)

var (
	cmdTable = map[string]string{
		"chan":                "chan",
		"temperature-control": "tec:out",
		"laser-output":        "las:out",
		"laser-current":       "las:ldi"}
)

func badMethod(w http.ResponseWriter, r *http.Request) {
	fstr := fmt.Sprintf("%s queried %s with bad method %s, must be either GET or POST", r.RemoteAddr, r.URL, r.Method)
	log.Println(fstr)
	http.Error(w, fstr, http.StatusMethodNotAllowed)
}

func stringToBool(s string) bool {
	return s == "1"
}

func boolToString(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

func floatToString(f float64) string {
	return strconv.FormatFloat(f, 'g', -1, 64)
}

// LDC3916 represents an LDC3916 laser diode controller
type LDC3916 struct {
	comm.RemoteDevice
	server.Server
}

// NewLDC3916 creates a new LDC3916 instance, which embeds both comm.RemoteDevice and server.Server
func NewLDC3916(addr, urlStem string) LDC3916 {
	rd := comm.NewRemoteDevice(addr, false)
	srv := server.NewServer(urlStem)
	ldc := LDC3916{RemoteDevice: rd}
	srv.RouteTable["chan"] = ldc.ChanDispatch
	srv.RouteTable["temperature-control"] = ldc.TempControlDispatch
	srv.RouteTable["laser-output"] = ldc.LaserOutputDispatch
	srv.RouteTable["laser-current"] = ldc.LaserCurrentDispatch
	srv.RouteTable["raw"] = ldc.RawDispatch
	ldc.Server = srv
	return ldc
}

// TxTermination overloads the value from RemoteDevice
func (ldc *LDC3916) TxTermination() byte {
	return termination
}

// RxTermination overloads the value from RemoteDevice
func (ldc *LDC3916) RxTermination() byte {
	return termination
}

// ChanDispatch handles (Get/Post) requests on /chan
func (ldc *LDC3916) ChanDispatch(w http.ResponseWriter, r *http.Request) {
	cmd := "chan"
	typ := "[]int"
	switch r.Method {
	case http.MethodGet:
		// build the command and format the response
		resp, err := ldc.processCommand(cmd, true, "")

		// response is going to look like 7 or 1,2,3
		httpResponder(resp, typ, err)(w, r)
	case http.MethodPost:
		obj := struct{ ints []int }{}
		err := json.NewDecoder(r.Body).Decode(obj)
		if err != nil {
			fstr := fmt.Sprintf("unable to decode int array from query.  Query must be a JSON request with \"ints\" field.  For a single channel, use a length-1 array.  %q", err)
			log.Println(fstr)
			http.Error(w, fstr, http.StatusBadRequest)
		}
		// we got the channel from the request, now set it on the device
		resp, err := ldc.processCommand(cmd, false, util.IntSliceToCSV(obj.ints))
		httpResponder(resp, typ, err)(w, r)
	default:
		badMethod(w, r)
	}
}

// TempControlDispatch handles (Get/Post) requests on /temperature-control
func (ldc *LDC3916) TempControlDispatch(w http.ResponseWriter, r *http.Request) {
	cmd := "temperature-control"
	typ := "bool"
	// this function is basically identical to ChanDispatch
	switch r.Method {
	case http.MethodGet:
		resp, err := ldc.processCommand(cmd, true, "")
		httpResponder(resp, typ, err)(w, r)
	case http.MethodPost:
		boo := server.BoolT{}
		err := json.NewDecoder(r.Body).Decode(boo)
		if err != nil {
			fstr := fmt.Sprintf("unable to decode boolean from query.  Query must be a JSON request with \"bool\" field.  %q", err)
			log.Println(fstr)
			http.Error(w, fstr, http.StatusBadRequest)
		}
		resp, err := ldc.processCommand(cmd, false, boolToString(boo.Bool))
		httpResponder(resp, typ, err)(w, r)
	default:
		badMethod(w, r)
	}
}

//LaserOutputDispatch handles (Get/Post) requests on /laser-output
func (ldc *LDC3916) LaserOutputDispatch(w http.ResponseWriter, r *http.Request) {
	cmd := "laser-output"
	typ := "bool"
	// this function is basically identical to tempcontroldispatch
	switch r.Method {
	case http.MethodGet:
		resp, err := ldc.processCommand(cmd, true, "")
		httpResponder(resp, typ, err)(w, r)
	case http.MethodPost:
		boo := server.BoolT{}
		err := json.NewDecoder(r.Body).Decode(boo)
		if err != nil {
			fstr := fmt.Sprintf("unable to decode boolean from query.  Query must be a JSON request with \"bool\" field.  %q", err)
			log.Println(fstr)
			http.Error(w, fstr, http.StatusBadRequest)
		}
		resp, err := ldc.processCommand(cmd, false, boolToString(boo.Bool))
		httpResponder(resp, typ, err)(w, r)
	default:
		badMethod(w, r)
	}
}

//LaserCurrentDispatch handles (Get/Post) requests on /laser-output
func (ldc *LDC3916) LaserCurrentDispatch(w http.ResponseWriter, r *http.Request) {
	cmd := "laser-current"
	typ := "float"
	// this function is basically identical to tempcontroldispatch
	switch r.Method {
	case http.MethodGet:
		resp, err := ldc.processCommand(cmd, true, "")
		httpResponder(resp, typ, err)(w, r)
	case http.MethodPost:
		float := server.FloatT{}
		err := json.NewDecoder(r.Body).Decode(float)
		if err != nil {
			fstr := fmt.Sprintf("unable to decode boolean from query.  Query must be a JSON request with \"bool\" field.  %q", err)
			log.Println(fstr)
			http.Error(w, fstr, http.StatusBadRequest)
		}
		resp, err := ldc.processCommand(cmd, false, floatToString(float.F64))
		httpResponder(resp, typ, err)(w, r)
	default:
		badMethod(w, r)
	}
}

//RawDispatch handles (Get/Post) requests on /laser-output
func (ldc *LDC3916) RawDispatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method must be POST", http.StatusMethodNotAllowed)
		return
	}
	str := server.StrT{}
	json.NewDecoder(r.Body).Decode(&str)
	resp, err := ldc.processCommand(str.Str, true, "")
	httpResponder(resp, "string", err)(w, r)
}

func (ldc *LDC3916) processCommand(cmd string, read bool, data string) (string, error) {
	cmd = cmdTable[cmd]
	if read {
		cmd = cmd + "?"
	}
	if data != "" {
		cmd = cmd + " " + data
	}
	err := ldc.Open()
	if err != nil {
		return "", err
	}
	defer ldc.Close()
	err = ldc.Send([]byte(cmd))
	if err != nil {
		return "", err
	}
	if read {
		r, err := ldc.Recv()
		if err != nil {
			return "", err
		}
		resp := string(r)
		return resp, nil
	}
	buf := make([]byte, 80)
	n, err := ldc.RemoteDevice.Conn.Read(buf)
	if err != nil {
		return "", err
	}
	return string(buf[:n]), nil
}

func httpResponder(data string, typ string, err error) http.HandlerFunc {
	// this function is fragile because we encode the type in a string instead
	// of using, say, types.BasicKind.  We do so because we need chan []int
	// and int slices are not a basic type
	if err != nil {
		return func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
	if data == "" {
		return func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}
	}
	var ret interface{}
	switch typ {
	case "[]int":
		s := strings.Split(string(data), ",")
		ints := make([]int, len(s))
		for idx, str := range s {
			v, err := strconv.Atoi(str)
			if err != nil {
				log.Fatal(err)
			}
			ints[idx] = v
		}
		if len(ints) == 1 {
			intt := ints[0]
			ret = struct{ int int }{intt}
		} else {
			ret = struct{ int []int }{ints}
		}
	case "bool":
		b := stringToBool(data)
		ret = struct{ bool bool }{b}
	case "float":
		f, err := strconv.ParseFloat(data, 64)
		if err != nil {
			return func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}
		ret = struct{ f64 float64 }{f}
	case "default":
		ret = struct{ str string }{data}
	}
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err := json.NewEncoder(w).Encode(ret)
		if err != nil {
			fstr := fmt.Sprintf("error encoding data to json state %q", err)
			log.Println(fstr)
			http.Error(w, fstr, http.StatusInternalServerError)
		}
		return
	}
}
