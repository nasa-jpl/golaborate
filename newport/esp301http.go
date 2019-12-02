package newport

import (
	"encoding/json"
	"fmt"
	"go/types"
	"log"
	"net/http"

	"github.jpl.nasa.gov/HCIT/go-hcit/server"

	"goji.io/pat"
)

// ESP301HTTPWrapper wraps ESP301 operation in an HTTP interface
type ESP301HTTPWrapper struct {
	// ESP is the underlying motion controller
	*ESP301

	// RouteTable is the map of Goji patterns to route handlers
	RouteTable server.RouteTable
}

// NewESP301HTTPWrapper returns a new wrapper with the route table populated
func NewESP301HTTPWrapper(esp *ESP301) ESP301HTTPWrapper {
	w := ESP301HTTPWrapper{ESP301: esp}
	rt := server.RouteTable{
		pat.Post("raw"):            w.Raw,
		pat.Post("single-cmd"):     w.JSONSingle,
		pat.Post("multi-cmd"):      w.JSONArray,
		pat.Get("cmd-list"):        w.CmdList,
		pat.Get("simple-pos-abs"):  w.GetPosAbs,
		pat.Post("simple-pos-abs"): w.SetPosAbs,
		pat.Post("simple-home"):    w.Home,
		pat.Get("errors"):          w.Errors,
	}
	w.RouteTable = rt
	return w
}

// Raw sends text to the ESP and returns the text it replies with.
// Do not include terminators, the server will take care of it for you
func (h *ESP301HTTPWrapper) Raw(w http.ResponseWriter, r *http.Request) {
	s := server.StrT{}
	err := json.NewDecoder(r.Body).Decode(&s)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp, err := h.ESP301.RawCommand(s.Str)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	b := append([]byte(resp), '\n')
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Length", string(len(b)))
	w.Write(b)
	return
}

// Errors reads the errors and returns them as a json [string] over HTTP
func (h *ESP301HTTPWrapper) Errors(w http.ResponseWriter, r *http.Request) {
	errors, err := h.ESP301.ReadErrors()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(errors)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	return
}

// Home homes an axis
func (h *ESP301HTTPWrapper) Home(w http.ResponseWriter, r *http.Request) {
	jcmd := JSONCommand{}
	err := json.NewDecoder(r.Body).Decode(&jcmd)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = h.ESP301.Home(jcmd.Axis)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// Wait waits for motion to cease
func (h *ESP301HTTPWrapper) Wait(w http.ResponseWriter, r *http.Request) {
	// this is a copy paste of wait instead of home
	jcmd := JSONCommand{}
	err := json.NewDecoder(r.Body).Decode(&jcmd)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = h.ESP301.Wait(jcmd.Axis)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	return
}

// GetPosAbs gets the absolute position of an axis
func (h *ESP301HTTPWrapper) GetPosAbs(w http.ResponseWriter, r *http.Request) {
	jcmd := JSONCommand{}
	err := json.NewDecoder(r.Body).Decode(&jcmd)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	f, err := h.ESP301.GetPos(jcmd.Axis)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	resp := server.HumanPayload{Float: f, T: types.Float64}
	resp.EncodeAndRespond(w, r)
	return
}

// SetPosAbs gets the absolute position of an axis
func (h *ESP301HTTPWrapper) SetPosAbs(w http.ResponseWriter, r *http.Request) {
	jcmd := JSONCommand{}
	err := json.NewDecoder(r.Body).Decode(&jcmd)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = h.ESP301.SetPosAbs(jcmd.Axis, jcmd.F64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	return
}

// JSONSingle handles singular commands over HTTP of JSONCommand type
func (h *ESP301HTTPWrapper) JSONSingle(w http.ResponseWriter, r *http.Request) {
	jcmd := &JSONCommand{}
	err := json.NewDecoder(r.Body).Decode(jcmd)
	defer r.Body.Close()
	if err != nil {
		fstr := fmt.Sprintf("error decoding JSON, request should have 3 fields; \"axis\", \"cmd\", \"f64\".  axis and f64 may be left blank.  %q", err)
		http.Error(w, fstr, http.StatusBadRequest)
		return
	}
	cmd, err := commandFromCmdOrAlias(jcmd.Cmd)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tele := makeTelegram(cmd, jcmd.Axis, jcmd.Write, jcmd.F64)
	err = h.ESP301.Open()
	if err != nil {
		fstr := fmt.Sprintf("error opening connection to motion controller %q", err)
		http.Error(w, fstr, http.StatusInternalServerError)
		return
	}
	defer h.ESP301.CloseEventually()
	resp, err := h.ESP301.SendRecv([]byte(tele))
	if err != nil {
		fstr := fmt.Sprintf("error communicating with motion controller %q.  Received response %q", err, resp)
		http.Error(w, fstr, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Length", string(len(resp)))
	w.Write(append(resp, byte('\n')))
}

// JSONArray handles arrays of commands over HTTP of JSONCommand type
func (h *ESP301HTTPWrapper) JSONArray(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, fstr, http.StatusInternalServerError)
		return
	}

	err = h.ESP301.Open()
	if err != nil {
		fstr := fmt.Sprintf("error opening connection to motion controller %q", err)
		http.Error(w, fstr, http.StatusInternalServerError)
		return
	}
	defer h.ESP301.CloseEventually()
	resp, err := h.ESP301.SendRecv([]byte(tele))
	if err != nil {
		fstr := fmt.Sprintf("error communicating with motion controller %q", err)
		http.Error(w, fstr, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Length", string(len(resp)))
	w.Write(append(resp, '\n'))
}

// CmdList returns a list of command objects which include:
// cmd (what Newport sees),
// alias (a friendier name you may use)
// description (a brief description)
// isReadOnly (whether the command is read-only or not)
// usesAxis (whether the command uses an axis or not)
func (h *ESP301HTTPWrapper) CmdList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(commands)
	if err != nil {
		fstr := fmt.Sprintf("json encoding error %q", err)
		http.Error(w, fstr, http.StatusInternalServerError)
		return
	}
	return
}
