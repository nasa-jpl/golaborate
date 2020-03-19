// Package keysight provides access to their oscilloscopes in Go
package keysight

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"time"
	"unsafe"

	"github.jpl.nasa.gov/HCIT/go-hcit/comm"
	"github.jpl.nasa.gov/HCIT/go-hcit/oscilloscope"
	"github.jpl.nasa.gov/HCIT/go-hcit/scpi"
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
	scpi.SCPI
}

// NewScope creates a new scope instance
func NewScope(addr string) *Scope {
	term := comm.Terminators{Tx: '\n', Rx: '\n'}
	rd := comm.NewRemoteDevice(addr, false, &term, nil)
	rd.Timeout = 24 * time.Hour
	return &Scope{scpi.SCPI{RemoteDevice: &rd, Handshaking: true}}
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
func (s *Scope) SetSampleRate(samplesPerSecond int) error {
	str := fmt.Sprintf("ACQuire:SRATe:ANALog %d", samplesPerSecond)
	return s.Write(str)
}

// GetSampleRate returns the sampling rate of the scope
func (s *Scope) GetSampleRate() (int, error) {
	return s.ReadInt("ACQuire:SRATe:ANAlog?")
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
	s.SCPI.RemoteDevice.Send([]byte(":WAVeform:DATA?"))
	var ret []byte
	buf := make([]byte, 9000) // as of 2020, even jumbo frames aren't bigger than this
	n, err := s.RemoteDevice.Conn.Read(buf)
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
	s.RemoteDevice.Lock()
	defer s.RemoteDevice.Unlock()
	if len(dataBuf) < nbytes { // this if may be removable
		for len(dataBuf) < nbytes {
			buf := make([]byte, 9000) // as of 2020, even jumbo frames aren't bigger than this
			n, err = s.RemoteDevice.Conn.Read(buf)
			s.RemoteDevice.LastComm = time.Now()
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
	ret.Data = make(map[string][]byte)
	ret.Scale = make(map[string]float64)
	ret.Offset = make(map[string]float64)
	// first, make sure the scope is sending data in our machine byte order
	if nativeEndian == binary.LittleEndian {
		byteCmd = "LSBFirst"
	} else {
		byteCmd = "MSBFirst"
	}
	fmt.Println("byte order")
	err := s.Write(":WAVeform:BYTeorder", byteCmd)
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
	fmt.Println("digi")
	fmt.Println(chunks)
	err = s.Write(chunks...)
	if err != nil {
		return ret, err
	}

	time.Sleep(time.Duration(timebase * 1e9))

	// now while we wait for ACQ to complete, we can get all
	// of the metadata
	fmt.Println("xinc")
	ret.DT, err = s.XIncrement()
	if err != nil {
		return ret, err
	}
	fmt.Println("unsigned")
	unsigned, err := s.ReadBool(":WAVeform:UNSigned?")
	if err != nil {
		return ret, err
	}
	if unsigned {
		ret.Dtype = "uint16"
	} else {
		ret.Dtype = "int16"
	}

	for i := 0; i < len(chanS); i++ {
		fmt.Println("source")
		// change the source so we can query for each channel
		err = s.Write(":WAVeform:SOURce", chanS[i])
		if err != nil {
			return ret, err
		}
		// get the vertical offset
		fmt.Println("yorigin")
		yoff, err := s.ReadFloat(":WAVeform:YORigin?")
		if err != nil {
			return ret, err
		}
		ret.Offset[channels[i]] = yoff

		// and the scale
		fmt.Println("yinc")
		yscale, err := s.ReadFloat(":WAVeform:YINCrement?")
		if err != nil {
			return ret, err
		}
		ret.Scale[channels[i]] = yscale
	}

	for i := 0; i < len(chanS); i++ {
		fmt.Println("source")
		err = s.Write(":WAVeform:SOURce", chanS[i])
		if err != nil {
			return ret, err
		}
		fmt.Println("data")
		buf, err := s.getBuffer()
		if err != nil {
			return ret, err
		}
		ret.Data[channels[i]] = buf
	}
	return ret, nil
}