// Package keysight provides access to their oscilloscopes in Go
package keysight

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unsafe"

	"github.jpl.nasa.gov/HCIT/go-hcit/comm"
)

var nativeEndian binary.ByteOrder

func init() {
	buf := [2]byte{}
	*(*uint16)(unsafe.Pointer(&buf[0])) = uint16(0xABCD)

	switch buf {
	case [2]byte{0xCD, 0xAB}:
		nativeEndian = binary.LittleEndian
	case [2]byte{0xAB, 0xCD}:
		nativeEndian = binary.BigEndian
	default:
		panic("Could not determine native endianness.")
	}
}

// Scope is an interface to a keysight oscilloscope
type Scope struct {
	*comm.RemoteDevice
}

// NewScope creates a new scope instance
func NewScope(addr string) *Scope {
	term := comm.Terminators{Tx: '\n', Rx: '\n'}
	rd := comm.NewRemoteDevice(addr, false, &term, nil)
	return &Scope{&rd}
}

// copied from agilent.go here to readBool
func (s *Scope) writeOnlyBus(cmds ...string) error {
	err := s.RemoteDevice.Open()
	if err != nil {
		return err
	}
	defer s.CloseEventually()
	str := strings.Join(cmds, " ")
	return s.RemoteDevice.Send([]byte(str))
}

func (s *Scope) readString(cmds ...string) (string, error) {
	s.writeOnlyBus(cmds...)
	resp, err := s.RemoteDevice.Recv()
	if err != nil {
		return "", err
	}
	return string(resp), nil
}

func (s *Scope) readFloat(cmds ...string) (float64, error) {
	s.writeOnlyBus(cmds...)
	resp, err := s.RemoteDevice.Recv()
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(string(resp), 64)
}

func (s *Scope) readBool(cmds ...string) (bool, error) {
	s.writeOnlyBus(cmds...)
	resp, err := s.RemoteDevice.Recv()
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(string(resp))
}

func (s *Scope) readInt(cmds ...string) (int, error) {
	s.writeOnlyBus(cmds...)
	resp, err := s.RemoteDevice.Recv()
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(string(resp))
}

// SetScale gets the vertical scale of the scope
func (s *Scope) SetScale(channel string, voltsFullScale float64) error {
	str := fmt.Sprintf(":CHANnel%s:RANGe %E", channel, voltsFullScale)
	return s.writeOnlyBus(str)
}

// GetScale returns the scale of the scope in volts full scale
func (s *Scope) GetScale(channel string) (float64, error) {
	str := fmt.Sprintf(":CHANnel%s:RANGe?", channel)
	return s.readFloat(str)
}

// SetTimebase sets the full timebase width of the scope in seconds
func (s *Scope) SetTimebase(fullWidth float64) error {
	str := fmt.Sprintf(":TIMebase:RANGe %E", fullWidth)
	return s.writeOnlyBus(str)
}

// GetTimebase returns the timebase width of the scope in seconds
func (s *Scope) GetTimebase() (float64, error) {
	str := ":TIMebase:RANge?"
	return s.readFloat(str)
}

// SetBandwidthLimit engages the bandwidth limit on the scope.
// If it is on, the noise is greatly reduced.
func (s *Scope) SetBandwidthLimit(channel string, on bool) error {
	var mnemonic string
	if on {
		mnemonic = "ON"
	} else {
		mnemonic = "OFF"
	}
	str := fmt.Sprintf(":CHANnel%s:BWLimit %s", channel, mnemonic)
	return s.writeOnlyBus(str)
}

// SetBitDepth configures the scope to use a given bit depth (vertical resolution)
func (s *Scope) SetBitDepth(bits int) error {
	str := fmt.Sprintf("ACQuire:HRESolution BITS%d", bits)
	return s.writeOnlyBus(str)
}

// GetBitDepth returns the number of bits used by the scope
func (s *Scope) GetBitDepth() (int, error) {
	str := fmt.Sprintf("ACQuire:HRESolution?")
	str, err := s.readString(str)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(str[4:]) // 4: -- original is "BITSxx"
}

// SetSampleRate sets the sampling rate of the scope in samples per second
func (s *Scope) SetSampleRate(samplesPerSecond int) error {
	str := fmt.Sprintf("ACQuire:SRATe:ANALog %d", samplesPerSecond)
	return s.writeOnlyBus(str)
}

// GetSampleRate returns the sampling rate of the scope
func (s *Scope) GetSampleRate() (int, error) {
	return s.readInt("ACQuire:SRATe:ANAlog?")
}

// SetAcqLength sets the total number of samples in an acquisition
func (s *Scope) SetAcqLength(points int) error {
	str := fmt.Sprintf("ACQuire:POINts:ANAlog %d", points)
	return s.writeOnlyBus(str)
}

// GetAcqLength returns the total number of points that will be acquired in a sequence
func (s *Scope) GetAcqLength() (int, error) {
	return s.readInt("ACQuire:POINts:ANAlog?")
}

// SetAcqMode sets the acquisition mode used by the scope
func (s *Scope) SetAcqMode(mode string) error {
	str := fmt.Sprintf("ACQuire:MODE %s", mode)
	return s.writeOnlyBus(str)
}

// GetAcqMode gets the acquisition mode used by the scope
func (s *Scope) GetAcqMode() (string, error) {
	return s.readString("ACQuire:MODE?")
}

// StartAcq triggers the beginning of acqisition on the scope
func (s *Scope) StartAcq() error {
	return s.writeOnlyBus(":DIGitize")
}

// SetStreaming puts the scope into or out of streaming mode
func (s *Scope) SetStreaming(on bool) error {
	var snip string
	if on {
		snip = "1"
	} else {
		snip = "0"
	}
	return s.writeOnlyBus(":WAVeform:STReaming", snip)
}

// GetStreaming returns true if the scope is in streaming mode for data transfers
func (s *Scope) GetStreaming() (bool, error) {
	return s.readBool(":WAVeform:STReaming?")
}

// DownloadData returns the data from the scope, after a StartAcq command
func (s *Scope) DownloadData() ([]int16, error) {
	var (
		byteCmd string
		ret     []int16
	)
	if nativeEndian == binary.LittleEndian {
		byteCmd = "LSBFirst"
	} else {
		byteCmd = "MSBFirst"
	}

	err := s.writeOnlyBus(":WAVeform:BYTeorder", byteCmd)
	if err != nil {
		return ret, err
	}

	err = s.SetStreaming(true)
	if err != nil {
		return ret, err
	}

	err = s.writeOnlyBus("WAVeform:DATA?")
	if err != nil {
		return ret, err
	}
	veryLargeBuffer, err := s.RemoteDevice.Recv()
	if err != nil {
		return ret, err
	}
	if len(veryLargeBuffer) < 2 {
		return ret, fmt.Errorf("response from scope was only %d bytes, expected >2", len(veryLargeBuffer))
	}
	if veryLargeBuffer[0] != '#' {
		return ret, fmt.Errorf("first byte in response from scope was %v, expected #", veryLargeBuffer[0])
	}
	if veryLargeBuffer[1] != '#' {
		return ret, fmt.Errorf("second byte in response from scope was %v, expected 0", veryLargeBuffer[1])
	}

	// now we do some slice hacking to convert the buffer to int16s
	secretlyint16s := veryLargeBuffer[2:]
	ary := []int16{}
	hdr := (*reflect.SliceHeader)(unsafe.Pointer(&secretlyint16s))
	hdr.Data = uintptr(unsafe.Pointer(&secretlyint16s[0]))
	hdr.Len = len(secretlyint16s) / 2
	hdr.Cap = cap(secretlyint16s) / 2
	return ary, nil
}
