package thermotek

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/nasa-jpl/golaborate/comm"
	"github.com/tarm/serial"
)

// T257P command primer
//
// messages are [SOC] [DeviceID] [Command Number] [CMD text] [CMD data] [Checksum] [TxTerm]
// resp.    are [SOR] [DeviceID] [Command Number] [Error] [CMD text] [CMD data] [Checksum] [TxTerm]
// SOC, SOR, TxTerm are constants in this package.  DeviceID is handled internally
// by the non-exported routines used to build up messages
//
// "read" commands begin with an r, while "set" commands begin with an s
//
// CamelCase is used in command names
// so for example, to read the sensor the command is 02rCtrlSen
//
//	[02] [r] [CtrlSen] are [Command Number] [read] [...resto f Text]
const (
	TxTerm   = '\r'
	soc      = '.' // start of command
	sor      = '#' // start of response
	cmdSize  = 24
	respSize = 27
)

var (
	errmap = map[byte]string{
		'0': "Command OK - No Errors",
		'1': "Checksum Error",
		'2': "Bad Command Number (Command Not Used)", // typo: lowercase u in manual
		'3': "Parameter/Data out of Bound",
		'4': "Message Length Error",
		'5': "Sensor/Feature not Configured or Used",
	}

	ErrMsgDoesNotStartWithSOR     = errors.New("chiller response did not begin with start-of-response (#)")
	ErrMsgDoesHaveCorrectDeviceID = errors.New("chiller response had device ID other than 01, or did not contain device ID at all")
	ErrChecksumMismatch           = errors.New("checksum mismatch")
)

// checksum computes the checksum as described in the manual
//
// the caller must ensure msg begins with SOC or SOR and device ID
//
// checksum returns a string encoding of the value
//
// TODO: this returning a string when the rest of this pkg uses bytes directly
// is a little silly
func checksum(msg []byte) [2]byte {
	// from manual:
	// Checksum field shall be two ASCII hexadecimal bytes representing the sum
	// of all preceding bytes (8 bit summation, no carry) of the command
	// starting with SOR. It is calculated and formatted the same way as the
	// command message checksum.

	var accumulator uint16
	for _, b := range msg {
		accumulator += uint16(b)
	}
	accumulator &= 0x00FF // kill off the upper byte
	return hexEncodeByte(byte(accumulator))
}

func frameMessage(msg []byte) []byte {
	// zero alloc
	var workspace [cmdSize]byte
	l := len(msg)
	if (l + 6) > cmdSize {
		panic(fmt.Sprintf("tried to frame message %s, which was longer than T257P can process", string(msg)))
	}
	workspace[0] = '.'
	workspace[1] = '0'
	workspace[2] = '1'
	copy(workspace[3:], msg)
	msg2 := workspace[:l+3]
	cs := checksum(msg2) // str -> bytes as ascii
	workspace[l+3] = cs[0]
	workspace[l+4] = cs[1]
	workspace[l+5] = TxTerm
	return workspace[:l+6] // slicing is end-exclusive
}

func checkAndUnframeResponse(resp []byte) ([]byte, error) {
	if resp[0] != sor {
		return nil, ErrMsgDoesNotStartWithSOR
	}
	if resp[1] != '0' || resp[2] != '1' {
		return nil, ErrMsgDoesHaveCorrectDeviceID
	}
	//                                                       0 to nine long
	//   0     1     2     3        4       5       6~13     14~(N-4)   N-3, N-2    N-1
	// [SOR] [DeviceID] [Command Number] [Error] [CMD text] [CMD data] [Checksum] [TxTerm]

	// ignore 3,4 command number
	errc := resp[5]
	if errc != '0' {
		if errc > '5' {
			return nil, fmt.Errorf("chiller returned error code %c, which was nonzero and unknown", errc)
		}
		return nil, errors.New(errmap[errc])
	}
	// // err is OK
	// // len of a data-less message is 1+2+2+1+8+2+1 = 17 bytes
	// if len(resp) > 17 {

	// }

	// compute and check the checksum
	cs := checksum(resp[:len(resp)-3])
	l := len(resp)
	if resp[l-3] != cs[0] || resp[l-2] != cs[1] {
		return nil, ErrChecksumMismatch
	}
	// pop the terminator and integers in the front of the message
	if resp[l-1] != TxTerm {
		return nil, errors.New("response from chiller did not end in a carriage return")
	}
	resp = resp[14 : l-3]
	return resp, nil
}

// makeSerConf makes a new serial.Config with correct parity, baud, etc, set.
func makeSerConf(addr string) *serial.Config {
	// T257P uses XON/XOFF flow control, which is not supported by tarm/serial
	// hopefully this still works
	return &serial.Config{
		Name:        addr,
		Baud:        9600,
		Size:        8,
		Parity:      serial.ParityNone,
		StopBits:    serial.Stop1,
		ReadTimeout: 3 * time.Second} // manual, under 3.1 timing
}

// T257P talks to the chiller of the same model name
type T257P struct {
	pool *comm.Pool
	// TODO: a semaphore to pace commands approp
}

// New257P creates a new T257P instance
func NewT257P(addr string) *T257P {
	maker := comm.SerialConnMaker(makeSerConf(addr))
	pool := comm.NewPool(1, time.Hour, maker)
	return &T257P{pool: pool}
}

func (t *T257P) WriteRead(msg []byte) ([]byte, error) {
	conn, err := t.pool.Get()
	if err != nil {
		return nil, err
	}
	defer func() { t.pool.ReturnWithError(conn, err) }()
	msg = frameMessage(msg)
	n, err := conn.Write(msg)
	if n != len(msg) {
		return nil, errors.New("T257P did not accept all bytes when sending command")
	}
	if err != nil {
		return nil, err
	}
	// zero alloc; backing array
	var workspace [respSize]byte
	buf := workspace[:]
	nTotal := 0
	MAXTRIES := respSize
	// device writes replies one byte at a time, creep through the response
	// scanning for terminator or maximum response length
	// TODO: a timeout would be a good idea, but also need
	// interrupt mechanism
	for i := 0; i < MAXTRIES; i++ {
		n, err := conn.Read(buf[nTotal:])
		nTotal += n
		if err != nil {
			return nil, err
		}
		if (buf[nTotal-1] == TxTerm) || nTotal == respSize {
			break
		}
	}
	return buf[:nTotal], nil
}

// return is in celcius (temps) or lpm (flow)
func (t *T257P) readFloat(cmd []byte) (float64, error) {
	resp, err := t.WriteRead(cmd)
	// resp is just the data field
	if err != nil {
		return 0, err
	}
	unframed, err := checkAndUnframeResponse(resp)
	if err != nil {
		return 0, err
	}
	f, err := strconv.ParseFloat(string(unframed), 64)
	f /= 10 // data is fixed precision, decimal place shifted one place
	return f, err
}

func (t *T257P) ReadTemperatureSetpoint() (float64, error) {
	return t.readFloat([]byte("03rSetTemp"))
}

func (t *T257P) ReadSupplyTemperature() (float64, error) {
	return t.readFloat([]byte("04rSupplyT"))
}

func (t *T257P) ReadExternalRTD() (float64, error) {
	return t.readFloat([]byte("05rExtRTD_"))
}

func (t *T257P) ReadExternalThermistor() (float64, error) {
	return t.readFloat([]byte("06rExtThrm"))
}

func (t *T257P) ReadAmbientTemperature() (float64, error) {
	return t.readFloat([]byte("08rAmbTemp"))
}

func (t *T257P) ReadFlowRate() (float64, error) {
	return t.readFloat([]byte("09rProsFlo"))
}

func (t *T257P) ReadHeatsinkTemperature(i int) (float64, error) {
	if i < 1 || i > 3 {
		return 0, fmt.Errorf("invalid heatsink index, valid values are 1-3, got %d", i)
	}
	return t.readFloat([]byte(fmt.Sprintf("67rHSnkTmp%d", i)))
}

func (t *T257P) ReadPlateTemperature(i int) (float64, error) {
	if i < 1 || i > 3 {
		return 0, fmt.Errorf("invalid plate index, valid values are 1-3, got %d", i)
	}
	return t.readFloat([]byte(fmt.Sprintf("69rPlatTmp%d", i)))
}

func (t *T257P) SetControlSensor(i int) error {
	if i < 0 || i > 3 {
		return fmt.Errorf("invalid control sensor index, valid values are 0-3, got %d", i)
	}
	cmd := fmt.Sprintf("16sCtrlSen%d", i)
	_, err := t.WriteRead([]byte(cmd))
	return err
}

func (t *T257P) SetControlTemperature(temp float64) error {
	if temp < -999.9 || temp > 999.9 {
		return fmt.Errorf("temperature must be in the range -999.9~999.9, got %f", temp)
	}
	tint := int(temp * 10)
	cmd := fmt.Sprintf("17sCtrlT__%+05d", tint)
	_, err := t.WriteRead([]byte(cmd))
	return err
}
