package newport

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.jpl.nasa.gov/HCIT/go-hcit/comm"
	"github.jpl.nasa.gov/HCIT/go-hcit/server"
	"github.jpl.nasa.gov/HCIT/go-hcit/util"
)

var (
	// XPSErrorCodes maps XPS error integers to strings
	XPSErrorCodes = map[int]string{
		0: "SUCCESS",

		-115: "HARDWARE FUNCTION NOT SUPPORTED",
		-113: "BOTH ENDS OF RUNS ACTIVATED",
		-112: "EXCITATION SIGNAL INITIALIZATION",
		-111: "GATHERING BUFFER FULL",
		-110: "NOT ALLOWED FOR GANTRY",
		-109: "NEED TO BE HOMED AT LEAST ONCE",
		-108: "SOCKET CLOSED BY ADMIN",
		-107: "NEED ADMINISTRATOR RIGHTS",
		-106: "WRONG USERNAME OR PASSWORD",
		-105: "SCALING CALIBRATION",
		-104: "PID TUNING INITIALIZATION",
		-103: "SIGNAL POINTS NOT ENOUGH",
		-102: "RELAY FEEDBACK TEST SIGNAL NOISY",
		-101: "RELAY FEEDBACK TEST NO OSCILLATION",
		-100: "INTERNAL ERROR",
		-99:  "FATAL EXTERNAL MODULE LOAD",
		-98:  "OPTIONAL EXTERNAL MODULE UNLOAD",
		-97:  "OPTIONAL EXTERNAL MODULE LOAD",
		-96:  "OPTIONAL EXTERNAL MODULE KILL",
		-95:  "OPTIONAL EXTERNAL MODULE EXECUTE",
		-94:  "OPTIONAL EXTERNAL MODULE FILE",

		-85: "HOME SEARCH GANTRY TOLERANCE ERROR",
		-83: "EVENT ID UNDEFINED",
		-82: "EVENT BUFFER FULL",
		-81: "ACTIONS NOT CONFIGURED",
		-80: "EVENTS NOT CONFIGURED",

		-75: "TRAJ TIME",
		-74: "READ FILE PARAMETER KEY",
		-73: "END OF FILE",
		-72: "TRAJ INIITALIZATION",
		-71: "MSG QUEUE",
		-70: "TRAJ FINAL VELOCITY",
		-69: "TRAJ ACC LIMIT",
		-68: "TRAJ VEL LIMIT",
		// no -67
		-66: "TRAJ EMPTY",
		-65: "TRAJ ELEM LINE",
		-64: "TRAJ ELEM SWEEP",
		-63: "TRAJ ELEM RADIUS",
		-62: "TRAJ ELEM TYPE",
		-61: "READ FILE",
		-60: "WRITE FILE",

		-51: "SPIN OUT OF RANGE",
		-50: "MOTOR INITIALIZATION ERROR",
		-49: "GROUP HOME SEARCH ZM ERROR",
		-48: "BASE VELOCITY",
		-47: "WRONG TCL TASKNAME",
		-46: "NOT ALLOWED BACKLASH",
		-45: "END OF RUN",
		-44: "SLAVE",
		-43: "GATHERING RUNNING",
		-42: "JOB OUT OF RANGE",
		-41: "SLAVE CONFIGURATION",
		-40: "MNEMO EVENT",
		-39: "NMEMO ACTION",
		-38: "TCL INTERPRETOR",
		-37: "TCL SCRIPT KILL",
		-36: "UNKNOWN TCL FILE",
		-35: "TRAVEL LIMITS",
		// no -34
		-33: "GROUP MOTION DONE TIMEOUT",
		-32: "GATHERING NOT CONFIGURED",
		-31: "HOME OUT OF RANGE",
		-30: "GATHERING NOT STARTED",
		-29: "MNEMOTYPEGATHERING",
		-28: "GROUP HOME SEARCH TIMEOUT",
		-27: "GROUP ABORT MOTION",
		-26: "EMERGENCY SIGNAL",
		-25: "FOLLOWING ERROR",
		-24: "UNCOMPATIBLE",
		-23: "POSITION COMPARE NOT SET",
		-22: "NOT ALLOWED ACTION",
		-21: "IN INITIALIZATION",
		-20: "FATAL INIT",
		-19: "GROUP NAME",
		-18: "POSITIONER NAME",
		-17: "PARAMETER OUT OF RANGE",
		-16: "WRONG TYPE UNSIGNEDINT",
		-15: "WRONG TYPE INT",
		-14: "WRONG TYPE DOUBLE",
		-13: "WRONG TYPE CHAR",
		-12: "WRONG TYPE BOOL",
		-11: "WRONG TYPE BIT WORD",
		-10: "WRONG TYPE",
		-9:  "WRONG PARAMETER NUMBER",
		-8:  "WRONG OBJECT TYPE",
		-7:  "WRONG FORMAT",
		// no -6
		-5: "POSITIONER ERROR",
		-4: "UNKNOWN COMMAND",
		-3: "STRING TOO LONG",
		-2: "TCP TIMEOUT",
		-1: "BUSY SOCKET",
		1:  "TCL INTERPRETOR ERROR",
	}
)

// XPSErr is a fancy Error() wrapper around error codes
type XPSErr int

// Error implements the error interface
func (e XPSErr) Error() string {
	if s, ok := XPSErrorCodes[int(e)]; ok {
		return fmt.Sprintf("%d - %s", e, s)
	}
	return fmt.Sprintf("%d - ERROR_UNKNOWN_TO_GO-HCIT", e)
}

// XPSError converts an error code to something that implements the error interface
func XPSError(code int) error {
	if code == 0 {
		return nil
	}
	return XPSErr(code)
}

// popError pulls the error code off of a raw response if it is present
// and returns the code as an int and the trimmed string
func popError(resp string) (int, string) {
	return 0, ""
}

// JSONGroupCommand is a primitive describing a command sent as JSON.
// CMD may either be a command (Command.Cmd) or an alias (Command.Alias)
// if Write is true, the data (F64) will be used.  If false, it will be ignored.
type JSONGroupCommand struct {
	Group string    `json:"group"`
	F64   []float64 `json:"f64"`
	Write bool      `json:"write"`
}

/*XPS represents an XPS series motion controller.

Note that the programming manual has a lot of socket numbers sprinkled around.
We do not see any here because those concern connection pools held by the
clients.  We only use a single connection, provided by comm.RemoteDevice.

While newport markets the XPS as a more versatile and consistent
(vis-a-vis communication) product than the older ESP line, this is not really
true in the author of this package's opinion.  For example, there is no function
that returns the number of positioners in a group, yet to move a positioner it
must belong to a group, and when you get the position of the group you must
supply the number of positioners to query for.  Consequently, a best practice
emerges to simply put each positioner in its own group, and not use the group
functionality at all.  This practice eliminates the ability to work with groups
the way they are used in most other motion controllers, which is a shame.
*/
type XPS struct {
	*comm.RemoteDevice
	server.Server
}

// NewXPS makes a new XPS instance
func NewXPS(addr, urlStem string) *XPS {
	rd := comm.NewRemoteDevice(addr, false, nil, makeSerConf(addr))
	srv := server.NewServer(urlStem)
	xps := XPS{RemoteDevice: &rd}
	srv.RouteTable["simple-pos-abs"] = xps.HTTPPos
	srv.RouteTable["simple-home"] = xps.HTTPHome
	xps.Server = srv
	return &xps
}

// GroupMoveAbsolute moves a group to an absolute position
func (xps *XPS) GroupMoveAbsolute(gid string, pos []float64) error {
	fstr := util.Float64SliceToCSV(pos, 'G', 9)
	cmd := fmt.Sprintf("GroupMoveAbsolute(%s,%s)", gid, fstr)
	fmt.Println(cmd)
	return nil
}

// GroupPositionCurrentGet gets the current absolute position of a group.
// Note that we hard-code "nbElements" to 1, since there is no way to
// query for that information, so the user will not know unless they are the
// only person who has ever assigned group elements on the controller.
func (xps *XPS) GroupPositionCurrentGet(gid string) ([]float64, error) {
	cmd := fmt.Sprintf("GroupPositionCurrentGet(%s, double *)", gid)
	fmt.Println(cmd)
	return []float64{0}, nil
}

// GroupHomeSearch homes each of the axes in a given group
func (xps *XPS) GroupHomeSearch(gid string) error {
	cmd := fmt.Sprintf("GroupHomeSearch(%s)", gid)
	fmt.Println(cmd)
	return XPSError(0)
}

// HTTPPos triggers GroupMoveAbsolute from an HTTP POST request with "group"
// (str) and "f64" ([]float) fields
// or GroupPositionCurrentGet from an HTTP GET request with "group" (str) field
func (xps *XPS) HTTPPos(w http.ResponseWriter, r *http.Request) {
	jcmd := JSONGroupCommand{}
	err := json.NewDecoder(r.Body).Decode(&jcmd)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodGet:
		pos, err := xps.GroupPositionCurrentGet(jcmd.Group)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s := struct {
			F64 []float64 `json:"f64"`
		}{pos}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(s)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return

	case http.MethodPost:
		err := xps.GroupMoveAbsolute(jcmd.Group, jcmd.F64)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		return

	}
}

// HTTPHome triggers GroupHomeSearch on the given group
func (xps *XPS) HTTPHome(w http.ResponseWriter, r *http.Request) {
	jcmd := JSONGroupCommand{}
	err := json.NewDecoder(r.Body).Decode(&jcmd)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = xps.GroupHomeSearch(jcmd.Group)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	return
}
