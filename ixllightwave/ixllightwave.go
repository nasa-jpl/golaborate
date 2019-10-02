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
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.jpl.nasa.gov/HCIT/go-hcit/util"
)

// the controller terminates with <CR> <NL> <END>
// it expects terminations of <NL> or <END> or <NL><END>
// we will use NL

const (
	// termination is the message termination used by the device
	termination = "\n"
)

func badMethod(w http.ResponseWriter, r *http.Request) {
	fstr := fmt.Sprintf("%s queried %s with bad method %s, must be either GET or POST", r.RemoteAddr, r.URL, r.Method)
	log.Println(fstr)
	http.Error(w, fstr, http.StatusMethodNotAllowed)
}

// IXLChan holds a single Chan field.  Not to be confused with golang channels.
// This represents a hardware channel and is an int.
type IXLChan struct {
	Chan []int `json:"chan"`
}

// EncodeAndRespond Encodes the data to JSON and writes to w.
// logs errors and replies with http.Error // status 500 on error
func (ch *IXLChan) EncodeAndRespond(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(ch)
	if err != nil {
		fstr := fmt.Sprintf("error encoding IXL Lightwave channel data to json state %q", err)
		log.Println(fstr)
		http.Error(w, fstr, http.StatusInternalServerError)
	}
	return
}

// IXLCurrent holds a single A field.  This is the current (/A/mperage) of the diode.  in mA.
type IXLCurrent struct {
	A float64 `json:"A"`
}

// EncodeAndRespond Encodes the data to JSON and writes to w.
// logs errors and replies with http.Error // status 500 on error
func (cur *IXLCurrent) EncodeAndRespond(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(cur)
	if err != nil {
		fstr := fmt.Sprintf("error encoding IXL Lightwave current data to json state %q", err)
		log.Println(fstr)
		http.Error(w, fstr, http.StatusInternalServerError)
	}
	return
}

// IXLTemp holds a single T field.  This is the temperature in Celcius.
type IXLTemp struct {
	T float64 `json:"T"`
}

// EncodeAndRespond Encodes the data to JSON and writes to w.
// logs errors and replies with http.Error // status 500 on error
func (temp *IXLTemp) EncodeAndRespond(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(temp)
	if err != nil {
		fstr := fmt.Sprintf("error encoding IXL Lightwave temperature data to json state %q", err)
		log.Println(fstr)
		http.Error(w, fstr, http.StatusInternalServerError)
	}
	return
}

// IXLBool holds a single B field (boolean).
type IXLBool struct {
	B bool `json:"B"`
}

// EncodeAndRespond Encodes the data to JSON and writes to w.
// logs errors and replies with http.Error // status 500 on error
func (on *IXLBool) EncodeAndRespond(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(on)
	if err != nil {
		fstr := fmt.Sprintf("error encoding IXL Lightwave Laser On/Off data to json state %q", err)
		log.Println(fstr)
		http.Error(w, fstr, http.StatusInternalServerError)
	}
	return
}

// LDC3916 represents an LDC3916 laser diode controller
type LDC3916 struct {
	Addr string
}

// GetChanAndReplyWithJSON reads the sensor over Conntype and responds with json-encoded TempHumic
func (ldc *LDC3916) GetChanAndReplyWithJSON(w http.ResponseWriter, r *http.Request) {
	ch, err := TCPGetChan(ldc.Addr)
	if err != nil {
		fstr := fmt.Sprintf("unable to read channel from controller sensor %+v, error %q", ldc, err)
		log.Println(fstr)
		http.Error(w, fstr, http.StatusInternalServerError)
		return
	}
	ch.EncodeAndRespond(w, r)
	log.Printf("%s checked channel %v", r.RemoteAddr, ch.Chan)
	return

}

// SetChanAndReplyWithJSON sets the channel and /DOES NOT/ reply with json, returning only 200/OK.
// The LDC does not send a reply, so we don't have one to send either...
func (ldc *LDC3916) SetChanAndReplyWithJSON(w http.ResponseWriter, r *http.Request) {
	data := &IXLChan{}
	err := json.NewDecoder(r.Body).Decode(data)
	if err != nil {
		fstr := fmt.Sprintf("unable to decode channel from query.  Channel must be formatted as an iterable, e.g. [7] for ch7.  %q", err)
		log.Println(fstr)
		http.Error(w, fstr, http.StatusBadRequest)
	} else {
		TCPSetChan(ldc.Addr, data.Chan)
		w.WriteHeader(200)
		log.Printf("%s set channel %v", r.RemoteAddr, data.Chan)
	}
	return
}

// ChanDispatch calls Get- or Set-ChanAndReplyWithJSON based on the request method (Get/Post)
func (ldc *LDC3916) ChanDispatch(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		ldc.GetChanAndReplyWithJSON(w, r)
	case http.MethodPost:
		ldc.SetChanAndReplyWithJSON(w, r)
	default:
		badMethod(w, r)
	}

}

// GetTempControlAndReplyWithJSON queries the LDC for the laser output status (on/off)
// and returns JSON with a boolean field "B" containing the reply.
func (ldc *LDC3916) GetTempControlAndReplyWithJSON(w http.ResponseWriter, r *http.Request) {
	// this function is almost identical to GetLaserControl[...]
	ctrl, err := TCPGetTempControl(ldc.Addr)
	if err != nil {
		fstr := fmt.Sprintf("error getting temperature control status %q", err)
		log.Println(err)
		http.Error(w, fstr, http.StatusInternalServerError)
	} else {
		ctrl.EncodeAndRespond(w, r)
		log.Printf("%s checked temperature control %v", r.RemoteAddr, ctrl.B)
	}
}

// SetTempControlAndReplyWithJSON sets the temperature control boolean and /DOES NOT/ reply with json, returning only 200/OK.
// The LDC does not send a reply, so we don't have one to send either...
func (ldc *LDC3916) SetTempControlAndReplyWithJSON(w http.ResponseWriter, r *http.Request) {
	data := &IXLBool{}
	err := json.NewDecoder(r.Body).Decode(data)
	if err != nil {
		fstr := fmt.Sprintf("unable to decode boolean from query.  query must be JSON with a field \"b\" containing the boolean. %q", err)
		log.Println(fstr)
		http.Error(w, fstr, http.StatusBadRequest)
	} else {
		TCPSetTempControl(ldc.Addr, data.B)
		w.WriteHeader(200)
		log.Printf("%s set temperature control %v", r.RemoteAddr, data.B)
	}
}

// TempControlDispatch calls Get- or Set-TempControlAndReplyWithJSON based on the request method (Get/Post)
func (ldc *LDC3916) TempControlDispatch(w http.ResponseWriter, r *http.Request) {
	// this function is basically identical to ChanDispatch
	switch r.Method {
	case http.MethodGet:
		ldc.GetTempControlAndReplyWithJSON(w, r)
	case http.MethodPost:
		ldc.SetTempControlAndReplyWithJSON(w, r)
	default:
		badMethod(w, r)
	}

}

// GetLaserControlAndReplyWithJSON queries the LDC for the laser output status (on/off)
// and returns JSON with a boolean field "B" containing the reply.
func (ldc *LDC3916) GetLaserControlAndReplyWithJSON(w http.ResponseWriter, r *http.Request) {
	ctrl, err := TCPGetLaserControl(ldc.Addr)
	if err != nil {
		fstr := fmt.Sprintf("error getting laser control status %q", err)
		log.Println(err)
		http.Error(w, fstr, http.StatusInternalServerError)
	} else {
		ctrl.EncodeAndRespond(w, r)
		log.Printf("%s got laser output %v", r.RemoteAddr, ctrl.B)
	}
}

// SetLaserControlAndReplyWithJSON sets the laser control boolean and /DOES NOT/ reply with json, returning only 200/OK.
// The LDC does not send a reply, so we don't have one to send either...
func (ldc *LDC3916) SetLaserControlAndReplyWithJSON(w http.ResponseWriter, r *http.Request) {
	data := &IXLBool{}
	err := json.NewDecoder(r.Body).Decode(data)
	if err != nil {
		fstr := fmt.Sprintf("unable to decode boolean from query.  query must be JSON with a field \"b\" containing the boolean. %q", err)
		log.Println(fstr)
		http.Error(w, fstr, http.StatusBadRequest)
	} else {
		TCPSetLaserControl(ldc.Addr, data.B)
		w.WriteHeader(200)
		log.Printf("%s set laser output %v", r.RemoteAddr, data.B)
	}
}

// LaserControlDispatch calls Get- or Set-LaserControlAndReplyWithJSON based on the request method (Get/Post)
func (ldc *LDC3916) LaserControlDispatch(w http.ResponseWriter, r *http.Request) {
	// this function is basically identical to ChanDispatch
	switch r.Method {
	case http.MethodGet:
		ldc.GetLaserControlAndReplyWithJSON(w, r)
	case http.MethodPost:
		ldc.SetLaserControlAndReplyWithJSON(w, r)
	default:
		badMethod(w, r)
	}

}

// GetLaserCurrentAndReplyWithJSON gets the laser current and replies with json containing field "A" holding the value.
func (ldc *LDC3916) GetLaserCurrentAndReplyWithJSON(w http.ResponseWriter, r *http.Request) {
	curr, err := TCPGetLaserCurrent(ldc.Addr)
	if err != nil {
		fstr := fmt.Sprintf("error getting laser current %q", err)
		log.Println(err)
		http.Error(w, fstr, http.StatusInternalServerError)
	} else {
		curr.EncodeAndRespond(w, r)
		log.Printf("%s got laser current %v mA", r.RemoteAddr, curr.A)
	}
}

// SetLaserCurrentAndReplyWithJSON sets the current and /DOES NOT/ reply with json, returning only 200/OK.
// The LDC does not send a reply, so we don't have one to send either...
func (ldc *LDC3916) SetLaserCurrentAndReplyWithJSON(w http.ResponseWriter, r *http.Request) {
	data := &IXLCurrent{}
	err := json.NewDecoder(r.Body).Decode(data)
	if err != nil {
		fstr := fmt.Sprintf("unable to decode current from query.  query must be JSON with a field \"A\" containing the current in mA. %q", err)
		log.Println(fstr)
		http.Error(w, fstr, http.StatusBadRequest)
	} else {
		TCPSetLaserCurrent(ldc.Addr, data.A)
		w.WriteHeader(200)
		log.Printf("%s set laser current %v mA", r.RemoteAddr, data.A)
	}
}

// LaserCurrentDispatch calls Get- or Set-LaserCurrentAndReplyWithJSON based on the request method (Get/Post)
func (ldc *LDC3916) LaserCurrentDispatch(w http.ResponseWriter, r *http.Request) {
	// this function is basically identical to ChanDispatch
	switch r.Method {
	case http.MethodGet:
		ldc.GetLaserCurrentAndReplyWithJSON(w, r)
	case http.MethodPost:
		ldc.SetLaserCurrentAndReplyWithJSON(w, r)
	default:
		badMethod(w, r)
	}

}

// RawRequest sends a raw ASCII request to the driver and get a raw response.
func (ldc *LDC3916) RawRequest(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fstr := fmt.Sprintf("unable to read command from request body %q", err)
		log.Println(fstr)
		http.Error(w, fstr, http.StatusBadRequest)
	}
	cmd := string(data)
	resp, err := TCPRawCmd(ldc.Addr, cmd)
	if err != nil {
		fstr := fmt.Sprintf("error from controller or reading response %q", err)
		log.Println(fstr)
		http.Error(w, fstr, http.StatusInternalServerError)
	} else {
		w.WriteHeader(200)
		io.WriteString(w, resp)
		log.Printf("%s sent raw command %s got response %s", r.RemoteAddr, cmd, resp)
	}

}

// BindRoutes binds HTTP routes to the methods of the LDC.  stem should not end in a slash.  Use "" for the index URL.
// ex: BindRoutes("/ldc") produces the following routes:
// /ldc/chan [GET/POST] channel setting for stateful commands
// /ldc/temperature-control [GET/POST] temperature control setting
// /ldc/laser-output [GET/POST] laser output control setting
// /ldc/laser-current [GET/POST] laser current in mA
// /ldc/raw [POST] a request with an ASCII text body and ASCII response
func (ldc *LDC3916) BindRoutes(stem string) {
	http.HandleFunc(stem+"/chan", ldc.ChanDispatch)
	http.HandleFunc(stem+"/temperature-control", ldc.TempControlDispatch)
	http.HandleFunc(stem+"/laser-output", ldc.LaserControlDispatch)
	http.HandleFunc(stem+"/laser-current", ldc.LaserCurrentDispatch)
	http.HandleFunc(stem+"/raw", ldc.RawRequest)
}

func tcpSetup(addr string) (net.Conn, error) {
	timeout := 3 * time.Second
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, err
	}
	deadline := time.Now().Add(timeout)
	conn.SetReadDeadline(deadline)
	conn.SetWriteDeadline(deadline)
	return conn, nil
}

func runWriteOnlyCommand(addr, cmd string) error {
	conn, err := tcpSetup(addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.Write([]byte(cmd + termination))
	if err != nil {
		return err
	}
	buf := make([]byte, 80)
	conn.Read(buf) // drain the buffer
	return nil
}

func readReply(conn net.Conn) ([]byte, error) {
	resp := make([]byte, 80) // the internal buffer on the LDC is 80 bytes, so we will use one of the same size
	n, err := conn.Read(resp)

	if err != nil {
		return resp, err
	}
	return resp[:n-1], nil
}

func tcpQueryBool(addr, cmd string) (bool, error) {
	conn, err := tcpSetup(addr)
	if err != nil {
		return false, nil
	}
	defer conn.Close()
	_, err = conn.Write([]byte(cmd + termination))
	if err != nil {
		return false, nil
	}

	resp, err := readReply(conn)
	if err != nil {
		return false, err
	}
	s := string(resp)
	var b bool
	if s == "1" {
		b = true
	} else {
		b = false
	}
	return b, nil
}

// TCPGetChan gets the current channel
func TCPGetChan(addr string) (IXLChan, error) {
	// open a tcp connection to the meter and send it our command
	conn, err := tcpSetup(addr)
	if err != nil {
		return IXLChan{}, err
	}
	defer conn.Close()
	_, err = conn.Write([]byte("chan?" + termination))
	if err != nil {
		return IXLChan{}, err
	}

	resp, err := readReply(conn)
	if err != nil {
		return IXLChan{}, err
	}
	s := strings.Split(string(resp), ",") // skip the last byte, it's a CR
	ints := make([]int, len(s))
	for idx, str := range s {
		v, err := strconv.Atoi(str)
		if err != nil {
			log.Fatal(err)
		}
		ints[idx] = v
	}
	return IXLChan{Chan: ints}, nil
}

// TCPSetChan sets the channel(s) for channel-sensitive commands
func TCPSetChan(addr string, chans []int) error {
	cmd := util.IntSliceToCSV(chans)
	cmd = "CHAN " + cmd
	return runWriteOnlyCommand(addr, cmd)
}

// TCPGetTempControl gets if temperature control is currently enabled (true).
func TCPGetTempControl(addr string) (IXLBool, error) {
	// open a tcp connection to the meter and send it our command
	cmd := "tec:out?"
	b, err := tcpQueryBool(addr, cmd)
	return IXLBool{B: b}, err
}

// TCPSetTempControl sets the temperature control on or off
func TCPSetTempControl(addr string, on bool) error {
	var cmd string
	if on {
		cmd = "1"
	} else {
		cmd = "0"
	}
	cmd = "tec:out " + cmd
	return runWriteOnlyCommand(addr, cmd)
}

// TCPGetLaserControl gets if the laser output is currently enabled (true)
func TCPGetLaserControl(addr string) (IXLBool, error) {
	cmd := "las:out?"
	b, err := tcpQueryBool(addr, cmd)
	return IXLBool{B: b}, err
}

// TCPSetLaserControl turns the laser on or off
func TCPSetLaserControl(addr string, on bool) error {
	// the code of this function is just about identical to TCPSetTempControl
	var cmd string
	if on {
		cmd = "1"
	} else {
		cmd = "0"
	}
	cmd = "las:out " + cmd
	return runWriteOnlyCommand(addr, cmd)
}

// TCPGetLaserCurrent gets the current laser current in mA.
func TCPGetLaserCurrent(addr string) (IXLCurrent, error) {
	// the body of this function is quite similar to TCPGetChan
	conn, err := tcpSetup(addr)
	if err != nil {
		return IXLCurrent{}, err
	}
	defer conn.Close()
	_, err = conn.Write([]byte("las:ldi?" + termination))
	if err != nil {
		return IXLCurrent{}, err
	}

	resp, err := readReply(conn)
	if err != nil {
		return IXLCurrent{}, err
	}
	f, err := strconv.ParseFloat(string(resp), 64)
	if err != nil {
		return IXLCurrent{}, err
	}
	return IXLCurrent{A: f}, nil

}

// TCPSetLaserCurrent sets the laser current in mA.
func TCPSetLaserCurrent(addr string, current float64) error {
	// the code of this function is just about identical to TCPSetTempControl
	cmd := "las:out " + strconv.FormatFloat(current, 'g', -1, 64)
	return runWriteOnlyCommand(addr, cmd)
}

// TCPRawCmd writes a raw command, appended the termination byte, and returns any response
func TCPRawCmd(addr, cmd string) (string, error) {
	log.Println([]byte(cmd))
	conn, err := tcpSetup(addr)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	_, err = conn.Write([]byte(cmd + termination))
	if err != nil {
		return "", err
	}

	resp, err := readReply(conn)
	if err != nil {
		return "", err
	}
	return string(resp), nil
}
