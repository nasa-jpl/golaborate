package aerotech

import (
	"fmt"
	"strings"
)

const (
	// OKCode is the first byte in the controller's response when the message
	// is acknowledged and response nominal
	OKCode = byte(37) // %

	// BadReqCode is the first byte in the controller's response when the message
	// was not understood
	BadReqCode = byte(33) // !

	// Terminator is the request terminator used
	Terminator = '\n'
)

type response struct {
	code byte
	body []byte
}

func (r response) isOK() bool {
	return r.code == OKCode
}

func (r response) string() string {
	return string(r.body)
}

func parse(raw []byte) response {
	if len(raw) < 2 {
		return response{}
	}
	var r response
	var v byte
	// scan for the ok/nok code.  Assume the last one belongs to us, if there
	// are multiple (e.g. unread responses)
	// it's ok to return something invalid if there was a "read" that was not
	// flushed, this should be considered unrecoverable.
	for {
		tmp := raw[0]
		if tmp == OKCode || tmp == BadReqCode {
			raw = raw[1:]
			v = tmp
		} else {
			break
		}
	}
	r.code = v
	// strip any terminators
	for raw[len(raw)-1] == Terminator {
		raw = raw[:len(raw)-1]
	}
	r.body = raw
	return r
}

// ErrBadResponse is generated when a bad response comes from the controller
type ErrBadResponse struct {
	resp string
}

func (e ErrBadResponse) Error() string {
	return fmt.Sprintf("bad response, OK returns %%, got %s", e.resp)
}

// Status is the Aerotech AXISSTATUS bitfield
type Status int32

// TODO: Zeebo's design here is fantastic -- backport this to all the other
// bitfields in golab

func (s Status) Enabled() bool            { return (s>>0)&1 == 1 }
func (s Status) Homed() bool              { return (s>>1)&1 == 1 }
func (s Status) InPosition() bool         { return (s>>2)&1 == 1 }
func (s Status) MoveActive() bool         { return (s>>3)&1 == 1 }
func (s Status) AccelPhase() bool         { return (s>>4)&1 == 1 }
func (s Status) DecelPhase() bool         { return (s>>5)&1 == 1 }
func (s Status) PositionCapture() bool    { return (s>>6)&1 == 1 }
func (s Status) CurrentClamp() bool       { return (s>>7)&1 == 1 }
func (s Status) BrakeOutput() bool        { return (s>>8)&1 == 1 }
func (s Status) MotionIsCw() bool         { return (s>>9)&1 == 1 }
func (s Status) MasterSlaveControl() bool { return (s>>10)&1 == 1 }
func (s Status) CalActive() bool          { return (s>>11)&1 == 1 }
func (s Status) CalEnabled() bool         { return (s>>12)&1 == 1 }
func (s Status) JoystickControl() bool    { return (s>>13)&1 == 1 }
func (s Status) Homing() bool             { return (s>>14)&1 == 1 }
func (s Status) MasterSuppress() bool     { return (s>>15)&1 == 1 }
func (s Status) GantryActive() bool       { return (s>>16)&1 == 1 }
func (s Status) GantryMaster() bool       { return (s>>17)&1 == 1 }
func (s Status) AutofocusActive() bool    { return (s>>18)&1 == 1 }
func (s Status) CommandFilterDone() bool  { return (s>>19)&1 == 1 }
func (s Status) InPosition2() bool        { return (s>>20)&1 == 1 }
func (s Status) ServoControl() bool       { return (s>>21)&1 == 1 }
func (s Status) CwEOTLimit() bool         { return (s>>22)&1 == 1 }
func (s Status) CcwEOTLimit() bool        { return (s>>23)&1 == 1 }
func (s Status) HomeLimit() bool          { return (s>>24)&1 == 1 }
func (s Status) MarkerInput() bool        { return (s>>25)&1 == 1 }
func (s Status) HallAInput() bool         { return (s>>26)&1 == 1 }
func (s Status) HallBInput() bool         { return (s>>27)&1 == 1 }
func (s Status) HallCInput() bool         { return (s>>28)&1 == 1 }
func (s Status) SineEncoderError() bool   { return (s>>29)&1 == 1 }
func (s Status) CosineEncoderError() bool { return (s>>30)&1 == 1 }
func (s Status) ESTOPInput() bool         { return (s>>31)&1 == 1 }

func (s Status) Bit(label string) bool {
	label = strings.ToLower(label)
	switch label {
	case "enabled":
		return s.Enabled()
	case "homed":
		return s.Homed()
	case "inposition":
		return s.InPosition()
	case "moveactive":
		return s.MoveActive()
	case "accelphase":
		return s.AccelPhase()
	case "decelphase":
		return s.DecelPhase()
	case "positioncapture":
		return s.PositionCapture()
	case "currentclamp":
		return s.CurrentClamp()
	case "brakeoutput":
		return s.BrakeOutput()
	case "motioniscw":
		return s.MotionIsCw()
	case "masterslavecontrol":
		return s.MasterSlaveControl()
	case "calactive":
		return s.CalActive()
	case "calenabled":
		return s.CalEnabled()
	case "joystickcontrol":
		return s.JoystickControl()
	case "homing":
		return s.Homing()
	case "mastersuppress":
		return s.MasterSuppress()
	case "gantryactive":
		return s.GantryActive()
	case "gantrymaster":
		return s.GantryMaster()
	case "autofocusactive":
		return s.AutofocusActive()
	case "commandfilterdone":
		return s.CommandFilterDone()
	case "inposition2":
		return s.InPosition2()
	case "servocontrol":
		return s.ServoControl()
	case "cweotlimit":
		return s.CwEOTLimit()
	case "ccweotlimit":
		return s.CcwEOTLimit()
	case "homelimit":
		return s.HomeLimit()
	case "markerinput":
		return s.MarkerInput()
	case "hallainput":
		return s.HallAInput()
	case "hallbinput":
		return s.HallBInput()
	case "hallcinput":
		return s.HallCInput()
	case "sineencodererror":
		return s.SineEncoderError()
	case "cosineencodererror":
		return s.CosineEncoderError()
	case "estopinput":
		return s.ESTOPInput()
	default:
		panic("aerotech: bit queried not present in Status bitfield")
	}
}

// All returns a k:v map of all bits in the bitfield
func (s Status) All() map[string]bool {
	return map[string]bool{
		"Enabled":            s.Enabled(),
		"Homed":              s.Homed(),
		"InPosition":         s.InPosition(),
		"MoveActive":         s.MoveActive(),
		"AccelPhase":         s.AccelPhase(),
		"DecelPhase":         s.DecelPhase(),
		"PositionCapture":    s.PositionCapture(),
		"CurrentClamp":       s.CurrentClamp(),
		"BrakeOutput":        s.BrakeOutput(),
		"MotionIsCw":         s.MotionIsCw(),
		"MasterSlaveControl": s.MasterSlaveControl(),
		"CalActive":          s.CalActive(),
		"CalEnabled":         s.CalEnabled(),
		"JoystickControl":    s.JoystickControl(),
		"Homing":             s.Homing(),
		"MasterSuppress":     s.MasterSuppress(),
		"GantryActive":       s.GantryActive(),
		"GantryMaster":       s.GantryMaster(),
		"AutofocusActive":    s.AutofocusActive(),
		"CommandFilterDone":  s.CommandFilterDone(),
		"InPosition2":        s.InPosition2(),
		"ServoControl":       s.ServoControl(),
		"CwEOTLimit":         s.CwEOTLimit(),
		"CcwEOTLimit":        s.CcwEOTLimit(),
		"HomeLimit":          s.HomeLimit(),
		"MarkerInput":        s.MarkerInput(),
		"HallAInput":         s.HallAInput(),
		"HallBInput":         s.HallBInput(),
		"HallCInput":         s.HallCInput(),
		"SineEncoderError":   s.SineEncoderError(),
		"CosineEncoderError": s.CosineEncoderError(),
		"ESTOPInput":         s.ESTOPInput(),
	}
}
