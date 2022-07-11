package keysight

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"strconv"
	"time"
	"unsafe"

	"github.com/nasa-jpl/golaborate/comm"
	"github.com/nasa-jpl/golaborate/oscilloscope"
	"github.com/nasa-jpl/golaborate/scpi"
)

var jumboFrameSize = 9000

// Scope is an interface to a keysight oscilloscope
type Scope struct {
	scpi.SCPI
}

// NewScope creates a new scope instance
func NewScope(addr string) *Scope {
	maker := comm.BackingOffTCPConnMaker(addr, 1*time.Second)
	pool := comm.NewPool(1, time.Hour, maker)
	return &Scope{scpi.SCPI{Pool: pool, Handshaking: true}}
}

// SetScale gets the vertical scale of the scope
func (s *Scope) SetScale(channel string, voltsFullScale float64) error {
	str := fmt.Sprintf(":CHANnel%s:RANGe %E", channel, voltsFullScale)
	return s.Write(str)
}

// GetScale returns the scale of the scope in volts full scale
func (s *Scope) GetScale(channel string) (float64, error) {
	str := fmt.Sprintf(":CHANnel%s:RANGe?", channel)
	return s.ReadFloat(str)
}

// SetOffset sets the vertical offset of the scope
func (s *Scope) SetOffset(channel string, voltsOffZero float64) error {
	str := fmt.Sprintf(":CHANnel%s:OFFSet %E", channel, voltsOffZero)
	return s.Write(str)
}

// GetOffset returns the vertical offset of a channel on the scope
func (s *Scope) GetOffset(channel string) (float64, error) {
	str := fmt.Sprintf(":CHANnel%s:OFFset?", channel)
	return s.ReadFloat(str)
}

// SetTimebase sets the full timebase width of the scope in seconds
func (s *Scope) SetTimebase(fullWidth float64) error {
	str := fmt.Sprintf(":TIMebase:RANGe %E", fullWidth)
	return s.Write(str)
}

// GetTimebase returns the timebase width of the scope in seconds
func (s *Scope) GetTimebase() (float64, error) {
	str := ":TIMebase:RANge?"
	return s.ReadFloat(str)
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
	return s.Write(str)
}

// SetBitDepth configures the scope to use a given bit depth (vertical resolution)
func (s *Scope) SetBitDepth(bits int) error {
	str := fmt.Sprintf("ACQuire:HRESolution BITS%d", bits)
	return s.Write(str)
}

// GetBitDepth returns the number of bits used by the scope
func (s *Scope) GetBitDepth() (int, error) {
	str := fmt.Sprintf("ACQuire:HRESolution?")
	str, err := s.ReadString(str)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(str[4:]) // 4: -- original is "BITSxx"
}

// SetSampleRate sets the sampling rate of the scope in samples per second
func (s *Scope) SetSampleRate(samplesPerSecond float64) error {
	i := int(samplesPerSecond)
	str := fmt.Sprintf("ACQuire:SRATe:ANALog %d", i)
	return s.Write(str)
}

// GetSampleRate returns the sampling rate of the scope
func (s *Scope) GetSampleRate() (float64, error) {
	i, err := s.ReadInt("ACQuire:SRATe:ANAlog?")
	return float64(i), err
}

// SetAcqLength sets the total number of samples in an acquisition
func (s *Scope) SetAcqLength(points int) error {
	// ACQuire:POINts:ANAlog -> WAVform:POINts 2020-03-11 in lab w/ MSO7104A
	str := fmt.Sprintf("WAVform:POINts %d", points)
	return s.Write(str)
}

// GetAcqLength returns the total number of points that will be acquired in a sequence
func (s *Scope) GetAcqLength() (int, error) {
	return s.ReadInt("WAVform:POINts?")
}

// SetAcqMode sets the acquisition mode used by the scope
func (s *Scope) SetAcqMode(mode string) error {
	str := fmt.Sprintf("ACQuire:MODE %s", mode)
	return s.Write(str)
}

// GetAcqMode gets the acquisition mode used by the scope
func (s *Scope) GetAcqMode() (string, error) {
	return s.ReadString("ACQuire:MODE?")
}

// StartAcq triggers the beginning of acqisition on the scope
func (s *Scope) StartAcq() error {
	return s.Write(":DIGitize")
}

// SetStreaming puts the scope into or out of streaming mode
func (s *Scope) SetStreaming(on bool) error {
	var snip string
	if on {
		snip = "1"
	} else {
		snip = "0"
	}
	return s.Write(":WAVeform:STReaming", snip)
}

// GetStreaming returns true if the scope is in streaming mode for data transfers
func (s *Scope) GetStreaming() (bool, error) {
	return s.ReadBool(":WAVeform:STReaming?")
}

// XIncrement gets the time delta of the scope's data record
func (s *Scope) XIncrement() (float64, error) {
	return s.ReadFloat(":WAVeform:XINCrement?")
}

// getBuffer transfers the data buffer from the scope handling all internal details
func (s *Scope) getBuffer() ([]byte, error) {
	var ret []byte
	conn, err := s.Pool.Get()
	if err != nil {
		return ret, err
	}
	defer func() { s.Pool.ReturnWithError(conn, err) }()
	_, err = conn.Write(append([]byte(":WAVeform:DATA?"), '\n'))
	if err != nil {
		return ret, err
	}
	buf := make([]byte, jumboFrameSize)
	n, err := conn.Read(buf)
	if err != nil {
		return ret, err
	}
	if n < 2 {
		return ret, fmt.Errorf("response from scope was only %d bytes, expected >2", n)
	}
	if buf[0] != '#' {
		return ret, fmt.Errorf("first byte in response from scope was %v, expected #", buf[0])
	}
	nbytesText := int(buf[1]) - 48 // shift down by 48, ASCII->int
	upper := 2 + nbytesText
	dataBuf := buf[:n]
	nbytes, err := strconv.Atoi(string(dataBuf[2:upper]))
	if err != nil {
		return ret, err
	}
	dataBuf = dataBuf[upper:]
	if len(dataBuf) < nbytes { // this if may be removable
		for len(dataBuf) < nbytes {
			buf := make([]byte, jumboFrameSize)
			n, err = conn.Read(buf)
			if err != nil {
				return ret, err
			}
			dataBuf = append(dataBuf, buf[:n]...)
		}
	}
	// now we need to pop off the terminator
	dataBuf = dataBuf[:len(dataBuf)-1]
	return dataBuf, nil
}

// AcquireWaveform configures the settings on the scope to digitize a waveform
// and return the data as a Waveform object with all information
// needed to convert to appropriate volts and time
func (s *Scope) AcquireWaveform(channels []string) (oscilloscope.Waveform, error) {
	var (
		byteCmd string
		ret     oscilloscope.Waveform
	)
	ret.Channels = map[string]oscilloscope.Channel{}
	// first, make sure the scope is sending data in our machine byte order
	if nativeEndian == binary.LittleEndian {
		byteCmd = "LSBFirst"
	} else {
		byteCmd = "MSBFirst"
	}
	err := s.Write(":WAVeform:FORMAT WORD")
	if err != nil {
		return ret, err
	}
	err = s.Write(":WAVeform:BYTeorder", byteCmd)
	if err != nil {
		return ret, err
	}

	// now, trigger digitization
	chunks := []string{":DIGitize"}

	chanS := make([]string, len(channels))
	for i := 0; i < len(channels); i++ {
		str := "CHANnel" + channels[i]
		chunks = append(chunks, str)
		chanS[i] = str
	}

	// get how long to sleep
	timebase, err := s.GetTimebase()
	if err != nil {
		return ret, err
	}
	err = s.Write(chunks...)
	if err != nil {
		return ret, err
	}

	time.Sleep(time.Duration(timebase * 1e9))

	ret.DT, err = s.XIncrement()
	if err != nil {
		return ret, err
	}
	unsigned, err := s.ReadBool(":WAVeform:UNSigned?")
	if err != nil {
		return ret, err
	}

	for i := 0; i < len(chanS); i++ {
		// change the source so we can query for each channel
		err = s.Write(":WAVeform:SOURce", chanS[i])
		if err != nil {
			return ret, err
		}
		// get the vertical offset
		yoff, err := s.ReadFloat(":WAVeform:YORigin?")
		if err != nil {
			return ret, err
		}

		// and the scale
		yscale, err := s.ReadFloat(":WAVeform:YINCrement?")
		if err != nil {
			return ret, err
		}

		// reference, because old scopes...
		yref, err := s.ReadFloat(":WAVeform:YREFerence?")
		if err != nil {
			return ret, err
		}

		buf, err := s.getBuffer()
		if err != nil {
			return ret, err
		}
		ch := oscilloscope.Channel{Scale: yscale, Offset: yoff, Reference: yref}
		if unsigned {
			var ary []uint16
			hdr := (*reflect.SliceHeader)(unsafe.Pointer(&ary))
			hdr.Data = uintptr(unsafe.Pointer(&buf[0]))
			hdr.Len = len(buf) / 2
			hdr.Cap = cap(buf) / 2
			ch.Data = ary
		} else {
			var ary []int16
			hdr := (*reflect.SliceHeader)(unsafe.Pointer(&ary))
			hdr.Data = uintptr(unsafe.Pointer(&buf[0]))
			hdr.Len = len(buf) / 2
			hdr.Cap = cap(buf) / 2
			ch.Data = ary
		}
		ret.Channels[chanS[i]] = ch
	}
	return ret, nil
}
