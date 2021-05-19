package acromag

/*
#include "apcommon.h"
#include "AP235.h"
#include "shim235.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"unsafe"
)

// AP235 is an acromag 16-bit DAC of the same type
type AP235 struct {
	sync.Mutex

	cfg *C.struct_cblk235

	// cursors hold the index into buffer
	// that corresponds to the current sample offset of each channel
	cursor [16]int

	// sample_count holds the number of samples in a supplied waveform
	// on a per-channel basis
	sampleCount [16]int

	// buffers are the sample queues for each channel, owned by C
	buffer [16][]uint16

	// cptrs holds the pointers in C to be used to free the buffers later
	cptr [16]*C.short

	cScatterInfo *C.ulong

	// playingBack is a global indicator of whether playback
	// is happening
	playingBack bool

	// isWaveform is a fast check for whether each channel is used
	// for waveform playback
	isWaveform [16]bool
}

// NewAP235 creates a new instance and opens the connection to the DAC
func NewAP235(deviceIndex int) (*AP235, error) {
	var (
		o    AP235
		out  = &o
		addr *C.struct_mapap235
		cs   = C.CString("ap235_") // untyped constant in C needs enforcement in Go
	)
	defer C.free(unsafe.Pointer(cs))

	o.cfg = (*C.struct_cblk235)(C.malloc(C.sizeof_struct_cblk235))
	o.cfg.pIdealCode = cMkCopyOfIdealData(idealCode)

	// open the board, initialize it, get its address, and populate its config
	errC := C.APOpen(C.int(deviceIndex), &o.cfg.nHandle, cs)
	err := enrich(errC, "APOpen")
	if err != nil {
		return out, err
	}

	errC = C.APInitialize(o.cfg.nHandle)
	err = enrich(errC, "APInitialize")
	if err != nil {
		return out, err
	}
	errC = C.GetAPAddress235(o.cfg.nHandle, &addr)
	err = enrich(errC, "GetAPAddress")
	if err != nil {
		return out, err
	}
	o.cfg.brd_ptr = addr
	o.cfg.pAP = C.GetAP(o.cfg.nHandle)
	if o.cfg.pAP == nil {
		return out, fmt.Errorf("unable to get a pointer to the acropack module")
	}

	// assign the buffer pointer
	ptr := C.Setup_board_corrected_buffer(o.cfg)
	if ptr == nil {
		return out, errors.New("error reading calibration data from AP235")
	}
	o.cScatterInfo = ptr
	// binitialize and bAP are set in Setup_board, ditto for rwcc235
	return out, nil
}

// SetRange configures the output range of the DAC
// this function only returns an error if the range is not allowed
// rngS is specified as in ValidateOutputRange
func (dac *AP235) SetRange(channel int, rngS string) error {
	dac.Lock()
	defer dac.Unlock()
	rng, err := ValidateOutputRange(rngS)
	if err != nil {
		return err
	}
	Crng := C.int(rng)
	dac.cfg.opts._chan[C.int(channel)].Range = Crng
	dac.sendCfgToBoard(channel)
	return nil
}

// GetRange returns the output range of the DAC in volts.
// The error value is always nil; the API looks
// this way for symmetry with Set
func (dac *AP235) GetRange(channel int) (string, error) {
	Crng := dac.cfg.opts._chan[C.int(channel)].Range
	return FormatOutputRange(OutputRange(Crng)), nil
}

// SetPowerUpVoltage configures the voltage set on the DAC at power up
// The error is only non-nil if the scale is invalid
func (dac *AP235) SetPowerUpVoltage(channel int, scale OutputScale) error {
	dac.Lock()
	defer dac.Unlock()
	if scale < ZeroScale || scale > FullScale {
		return fmt.Errorf("output scale %d is not allowed", scale)
	}
	dac.cfg.opts._chan[C.int(channel)].PowerUpVoltage = C.int(scale)
	dac.sendCfgToBoard(channel)
	return nil
}

// GetPowerUpVoltage retrieves the voltage of the DAC at power up.
// the error is always nil
func (dac *AP235) GetPowerUpVoltage(channel int) (OutputScale, error) {
	Cpwr := dac.cfg.opts._chan[C.int(channel)].PowerUpVoltage
	return OutputScale(Cpwr), nil
}

// SetClearVoltage sets the voltage applied at the output when the device is cleared
// the error is only non-nil if the voltage is invalid
func (dac *AP235) SetClearVoltage(channel int, scale OutputScale) error {
	dac.Lock()
	defer dac.Unlock()
	if scale < ZeroScale || scale > FullScale {
		return fmt.Errorf("output scale %d is not allowed", scale)
	}
	dac.cfg.opts._chan[C.int(channel)].ClearVoltage = C.int(scale)
	dac.sendCfgToBoard(channel)
	return nil
}

// GetClearVoltage gets the voltage applied at the output when the device is cleared
// The error is always nil
func (dac *AP235) GetClearVoltage(channel int) (OutputScale, error) {
	Cpwr := dac.cfg.opts._chan[C.int(channel)].ClearVoltage
	return OutputScale(Cpwr), nil
}

// SetOverTempBehavior sets the behavior of the device when an over temp
// is detected.  Shutdown == true -> shut down the board on overtemp
// the error is always nil
func (dac *AP235) SetOverTempBehavior(channel int, shutdown bool) error {
	dac.Lock()
	defer dac.Unlock()
	i := 0
	if shutdown {
		i = 1
	}
	dac.cfg.opts._chan[C.int(channel)].ThermalShutdown = C.int(i)
	dac.sendCfgToBoard(channel)
	return nil
}

// GetOverTempBehavior returns true if the device will shut down when over temp
// the error is always nil
func (dac *AP235) GetOverTempBehavior(channel int) (bool, error) {
	Cint := dac.cfg.opts._chan[C.int(channel)].ThermalShutdown
	return Cint == 1, nil
}

// SetOverRange configures if the DAC is allowed to exceed output limits by 5%
// allowed == true allows the DAC to operate slightly beyond limits
// the error is always nil
func (dac *AP235) SetOverRange(channel int, allowed bool) error {
	dac.Lock()
	defer dac.Unlock()
	i := 0
	if allowed {
		i = 1
	}
	dac.cfg.opts._chan[C.int(channel)].OverRange = C.int(i)
	dac.sendCfgToBoard(channel)
	return nil
}

// GetOverRange returns true if the DAC output is allowed to exceed nominal by 5%
// the error is always nil
func (dac *AP235) GetOverRange(channel int) (bool, error) {
	Cint := dac.cfg.opts._chan[C.int(channel)].OverRange
	return Cint == 1, nil
}

// SetTriggerMode configures the DAC for a given triggering mode
// the error is only non-nil if the trigger mode is invalid
func (dac *AP235) SetTriggerMode(channel int, triggerMode string) error {
	dac.Lock()
	defer dac.Unlock()
	tm, err := ValidateTriggerMode(triggerMode)
	if err != nil {
		return err
	}
	dac.cfg.opts._chan[C.int(channel)].TriggerSource = C.int(tm)
	opMode, _ := dac.GetOperatingMode(channel)
	dac.sendCfgToBoard(channel)
	if opMode == "waveform" {
		if (triggerMode != "external") && (triggerMode != "timer") {
			return ErrIncompatibleOperatingTrigger
		}
	}
	return nil
}

// GetTriggerMode retrieves the current triggering mode
// the error is always nil
func (dac *AP235) GetTriggerMode(channel int) (string, error) {
	tm := dac.cfg.opts._chan[C.int(channel)].TriggerSource
	return FormatTriggerMode(TriggerMode(tm)), nil
}

// SetTriggerDirection if the DAC's trigger is input (false) or output (true)
// the error is always nil.
func (dac *AP235) SetTriggerDirection(b bool) error {
	dac.Lock()
	defer dac.Unlock()
	var i int // init to zero value, false->0
	if b {
		i = 1
	}
	dac.cfg.TriggerDirection = C.uint32_t(i)
	// dac.sendCfgToBoard() TODO: need to send to board?
	return nil
}

// GetTriggerDirection returns true if the DAC's trigger is output, false if it is input
// the error is always nil
func (dac *AP235) GetTriggerDirection() (bool, error) {
	ci := dac.cfg.TriggerDirection
	var b bool
	if ci == 1 {
		b = true
	}
	return b, nil
}

// SetOperatingMode changes the operating mode of the DAC.
//
// Valid modes are 'single', 'waveform'.
//
// a non-nil error will be generated if the triggering mode
// for this channel is incomaptible.  The config change will
// still be made.
// err should be checked on the later of the two calls to
// SetOperatingMode and SetTriggerMode
func (dac *AP235) SetOperatingMode(channel int, mode string) error {
	dac.Lock()
	defer dac.Unlock()
	o, err := ValidateOperatingMode(mode)
	if err != nil {
		return err
	}
	dac.cfg.opts._chan[C.int(channel)].OpMode = C.int(o)
	trigger, _ := dac.GetTriggerMode(channel)
	dac.sendCfgToBoard(channel)
	if mode == "waveform" {
		dac.isWaveform[channel] = true
		if (trigger != "external") && (trigger != "timer") {
			return ErrIncompatibleOperatingTrigger
		}
	}
	dac.isWaveform[channel] = false
	return nil
}

// GetOperatingMode retrieves whether the DAC is in single sample or waveform mode
func (dac *AP235) GetOperatingMode(channel int) (string, error) {
	modeC := dac.cfg.opts._chan[C.int(channel)].OpMode
	return FormatOperatingMode(OperatingMode(modeC)), nil
}

// SetClearOnUnderflow configures the DAC to clear output on an underflow if true
// the error is always nil
func (dac *AP235) SetClearOnUnderflow(channel int, b bool) error {
	dac.Lock()
	defer dac.Unlock()
	var i int // init to zero value, false->0
	if b {
		i = 1
	}
	dac.cfg.opts._chan[C.int(channel)].UnderflowClear = C.int(i)
	dac.sendCfgToBoard(channel)
	return nil
}

// GetClearOnUnderflow configures the DAC to clear output on an underflow if true
// the error is always nil
func (dac *AP235) GetClearOnUnderflow(channel int) (bool, error) {
	ci := dac.cfg.opts._chan[C.int(channel)].UnderflowClear
	var b bool
	if ci == 1 {
		b = true
	}
	return b, nil
}

// SetOutputSimultaneous configures the DAC to simultaneous mode or async mode
// this function will always return nil.
func (dac *AP235) SetOutputSimultaneous(channel int, simultaneous bool) error {
	dac.Lock()
	defer dac.Unlock()
	sim := 0
	if simultaneous {
		sim = 1
	}
	// opts.chan -> opts._chan; cgo rule to replace go identifier
	dac.cfg.opts._chan[C.int(channel)].UpdateMode = C.int(sim)
	dac.sendCfgToBoard(channel)
	return nil
}

// GetOutputSimultaneous returns true if the DAC is in simultaneous output mode
// the error value is always nil
func (dac *AP235) GetOutputSimultaneous(channel int) (bool, error) {
	i := int(dac.cfg.opts._chan[C.int(channel)].UpdateMode)
	return i == 1, nil
}

// SetTimerPeriod sets the timer period,
// the time between repetitions of the timer clock
//
// there are two threshholds: 9920 ns, below which
// the DAC cannot settle to better than 1LSB
// before the next command and 19840 ns, below which
// the DAC cannot be fed data for all sixteen channels
// in parallel.
func (dac *AP235) SetTimerPeriod(nanoseconds uint32) error {
	dac.Lock()
	defer dac.Unlock()
	tdiv := nanoseconds / 32
	dac.cfg.TimerDivider = C.uint32_t(tdiv)
	if tdiv < 310 { // minimum recommended value from Acromag, based on DAC settling time
		return ErrTimerTooFast
	}
	if tdiv < 620 {
		return errors.New("timer too fast for transfer to DAC to keep up if all channels used; still accepted")
	}
	return nil
}

// GetTimerPeriod retrieves the timer period in nanoseconds
//
// the error is always nil
func (dac *AP235) GetTimerPeriod() (uint32, error) {
	return uint32(dac.cfg.TimerDivider) * 32, nil
}

// sendCfgToBoard updates the configuration on the board
func (dac *AP235) sendCfgToBoard(channel int) {
	C.cnfg235(dac.cfg, C.int(channel))
	return
}

// Output writes a voltage to a channel.
// the error is only non-nil if the value is out of range
func (dac *AP235) Output(channel int, voltage float64) error {
	// TODO: look into cd235 C function
	// this is a hack to improve code reuse, no need to allocate slices here
	vB := []float64{voltage}
	vU := []uint16{0}
	dac.calibrateData(channel, vB, vU)
	return dac.OutputDN16(channel, vU[0])
}

// OutputDN16 writes a value to the board in DN.
//
// if the channel is set up for waveform mode, an error is generated.
// otherwise, it is nil.
func (dac *AP235) OutputDN16(channel int, value uint16) error {
	dac.Lock()
	defer dac.Unlock()
	if dac.isWaveform[channel] {
		return ErrIncompatibleWaveform
	}
	// going to round trip, since we want to use the DAC in calibrated mode
	// convert value to a f64
	rng, _ := dac.GetRange(channel)
	min, max := RangeToMinMax(rng)
	step := (max - min) / 65535
	fV := []float64{min + step*float64(value)}

	// set FIFO configuration for this channel to 1 sample
	cCh := C.int(channel)
	dac.cfg.SampleCount[cCh] = 1
	ptr := &dac.cfg.pcor_buf[cCh][0]
	ptr2 := &dac.cfg.pcor_buf[cCh][1]
	dac.cfg.current_ptr[cCh] = ptr
	dac.cfg.head_ptr[cCh] = ptr
	dac.cfg.tail_ptr[cCh] = ptr2
	C.cd235(dac.cfg, C.int(channel), (*C.double)(&fV[0]))
	C.fifowro235(dac.cfg, cCh)
	return nil
}

// OutputMulti writes voltages to multiple output channels.
// the error is non-nil if any of these conditions occur:
//	1.  A blend of output modes (some simultaneous, some immediate)
//  2.  A command is out of range
//  3.  A channel is set up for waveform playback
//
// if an error is encountered in case 2, the output buffer of the DAC may be
// partially updated from proceeding valid commands.  No invalid values escape
// to the DAC output.
//
// The device is flushed after writing if the channels are simultaneous output.
//
// passing zero length slices will cause a panic.  Slices must be of equal length.
func (dac *AP235) OutputMulti(channels []int, voltages []float64) error {
	// how this is different to AP236:
	// AP236 is immediate output.  Write output -> it happens.
	// AP235 is waveform and has three triggering modes for each
	// channel:
	// 1.  software
	// 2.  timer
	// 3.  exterinal input
	// ensure channels are homogeneous
	sim, _ := dac.GetOutputSimultaneous(channels[0])
	for i := 0; i < len(channels); i++ { // old for is faster than range, this code may be hot
		tm, _ := dac.GetTriggerMode(channels[i])
		if tm != "software" {
			return fmt.Errorf("trigger mode must be software.  Channel %d was %s",
				channels[i], tm)
		}
		sim2, _ := dac.GetOutputSimultaneous(channels[i])
		if sim2 != sim {
			return fmt.Errorf("mixture of output modes used, must be homogeneous.  Channel %d != channel %d",
				channels[i], channels[0])
		}
		if dac.isWaveform[channels[i]] {
			return ErrIncompatibleWaveform
		}
	}

	for i := 0; i < len(channels); i++ {
		err := dac.Output(channels[i], voltages[i])
		if err != nil {
			return fmt.Errorf("channel %d voltage %f: %w", channels[i], voltages[i], err)
		}
	}
	if sim {
		dac.Flush()
	}
	return nil
}

// OutputMultiDN16 is equivalent to OutputMulti, but with DNs instead of volts.
// see the docstring of OutputMulti for more information.
func (dac *AP235) OutputMultiDN16(channels []int, uint16s []uint16) error {
	// how this is different to AP236:
	// AP236 is immediate output.  Write output -> it happens.
	// AP235 is waveform and has three triggering modes for each
	// channel:
	// 1.  software
	// 2.  timer
	// 3.  exterinal input
	// ensure channels are homogeneous
	sim, _ := dac.GetOutputSimultaneous(channels[0])
	for i := 0; i < len(channels); i++ { // old for is faster than range, this code may be hot
		tm, _ := dac.GetTriggerMode(channels[i])
		if tm != "software" {
			return fmt.Errorf("trigger mode must be software.  Channel %d was %s",
				channels[i], tm)
		}
		sim2, _ := dac.GetOutputSimultaneous(channels[i])
		if sim2 != sim {
			return fmt.Errorf("mixture of output modes used, must be homogeneous.  Channel %d != channel %d",
				channels[i], channels[0])
		}
		if dac.isWaveform[channels[i]] {
			return ErrIncompatibleWaveform
		}
	}

	for i := 0; i < len(channels); i++ {
		err := dac.OutputDN16(channels[i], uint16s[i])
		if err != nil {
			return fmt.Errorf("channel %d DN %d: %w", channels[i], uint16s[i], err)
		}
	}
	if sim {
		dac.Flush()
	}
	return nil
}

// Flush writes any pending output values to the device
func (dac *AP235) Flush() {
	C.simtrig235(dac.cfg)
}

// StartWaveform starts waveform playback on all waveform channels
// the error is only non-nil if playback is already occuring
func (dac *AP235) StartWaveform() error {
	dac.Lock()
	defer dac.Unlock()
	if dac.playingBack {
		return errors.New("AP235 is already playing back a waveform")
	}
	go dac.serviceInterrupts()
	dac.playingBack = true
	C.start_waveform(dac.cfg)
	return nil
}

// StopWaveform stops playback on all channels.
// the error is non-nil only if playback is not occuring
func (dac *AP235) StopWaveform() error {
	dac.Lock()
	defer dac.Unlock()
	if !dac.playingBack {
		return errors.New("AP235 is not playing back a waveform")
	}
	dac.playingBack = false
	C.stop_waveform(dac.cfg)
	return nil
}

// need software reset?  drvr235.c, L475

// calibrateData converts a f64 value to uint16.  This is basically cd235
// len(buffer) shall == len(volts)
func (dac *AP235) calibrateData(channel int, volts []float64, buffer []uint16) {
	// see AP235 manual (PDF), page 68
	cCh := C.int(channel)
	rngS, _ := dac.GetRange(channel)    // err always nil
	rng, _ := ValidateOutputRange(rngS) // err always nil
	gainCoef := 1 + float64(dac.cfg.ogc235[cCh][rng][gain])/(65535*16)
	slopeCoef := float64(dac.cfg.pIdealCode[rng][idealSlope])
	off := float64(dac.cfg.pIdealCode[rng][idealZeroBTC]) + float64(dac.cfg.ogc235[channel][rng][offset])/16
	gain := gainCoef * slopeCoef
	min := float64(dac.cfg.pIdealCode[rng][clipLo])
	max := float64(dac.cfg.pIdealCode[rng][clipHi])
	for i := 0; i < len(volts); i++ {
		in := volts[i]
		out := gain*in + off
		if out > max {
			out = max
		} else if out < min {
			out = min
		}
		// buffer[i] = int16(out) ^ 0x8000 // or with 0x8000 per the manual
		buffer[i] = uint16(out + 0x8000)
	}
}

// PopulateWaveform populates the waveform table for a given channel
// the error is only non-nil if the DAC is currently playing back a waveform
func (dac *AP235) PopulateWaveform(channel int, data []float64) error {
	// need to:
	// 1) convert f64 => uint16
	// 2) handle buffer, cursor, sample count
	//    sampleCount in Go, not SampleCount in C (which is <= 4096)
	// 3) free old buffers if this isn't the first time we're populating
	// 5) put the channel in FIFO_DMA mode
	// 4) do the first dma transfer
	// we do not start the background thread until waveform playback starts
	// since we only want to start the one thread, not one per channel.

	// create a buffer long enough to hold the waveform in uint16s
	if dac.playingBack {
		return errors.New("AP235 cannot change waveform table during playback")
	}

	err := dac.SetOperatingMode(channel, "waveform")
	if err != nil {
		return err // err is beneign, but force users to reconfigure DAC first
	}
	l := len(data)
	buf, cptr, err := cMkarrayU16(l)
	if err != nil {
		return err
	}

	err = dac.Clear(channel)
	if err != nil {
		return err // err is beneign, but dump the buffer first
	}
	if dac.cptr[channel] != nil {
		// free old buffer and replace
		C.aligned_free(unsafe.Pointer(dac.cptr[channel]))
	}
	dac.cptr[channel] = cptr

	// set the interrupt source for this channel (needed for transfer interrupt)
	dac.cfg.opts._chan[channel].InterruptSource = 1
	dac.sendCfgToBoard(channel) // need to make sure this value propagates to the FPGA

	// now convert each value to a u16 and update the buffer
	dac.calibrateData(channel, data, buf) // "moves" data->buf
	dac.sampleCount[channel] = l
	dac.cursor[channel] = 0
	dac.buffer[channel] = buf
	dac.cfg.head_ptr[channel] = (*C.short)(unsafe.Pointer(&dac.buffer[channel][0]))
	C.set_DAC_sample_addresses(dac.cfg, C.int(channel))
	dac.doTransfer(channel)
	return nil
}

// serviceInterrupts should be run as a background goroutine; it handles
// interrupts from the DAC to keep it fed
func (dac *AP235) serviceInterrupts() {
	// the minimum recommended timer period is 0x136
	// which is (310 * 32 ns) = 9.9us
	// so this loop could happen as frequently as
	// 9.9us * 2048 samples = 20 ms
	// it's not all that hot after all.
	//
	// the above was for DMA, and is true
	// however, now we are considering the non-DMA case
	// where it is 1us/sample transfer time
	// so the interrupt could come and we need
	// 2048*16 = 32768 samples = 32768 or more us
	// per interrupt, given a period of only about 20 ms.
	// this is not viable.
	// suggest to add an error to the timer period
	// function for periods < 2x the recommended limit
	// which is ~50kHz
	dac.Lock()
	C.enable_interrupts(dac.cfg)
	dac.Unlock()
	for {
		// fetch_status blocks for an extended period, so we will hold the lock
		// for an extended period.  At 5k samples per second and 2048 samples
		// per channel, that could be 500 ms (an eternitity for real time)
		// this is a nasty pickle.
		//
		// it isn't solved here.  You might be okay if you run a waveform and
		// try to talk to the DAC on other channels at the same time.  It could
		// deadlock.  I don't know how to solve this problem in a way that makes
		// the waveform and any real time needs happy simultaneously.
		// TODO
		Cstatus := C.fetch_status(dac.cfg)
		status := uint(Cstatus)

		if status == 0 {
			return
		}
		// at least one channel requires updating
		for i := 0; i < 16; i++ { // i = channel index
			var mask uint = 1 << i
			if (mask & status) != 0 {
				dac.Lock()
				dac.doTransfer(i)
				dac.Unlock()
			}
		}
		C.refresh_interrupt(dac.cfg, Cstatus)
	}
}

// Clear soft resets the DAC, clearing the output but not configuration
// the error is always nil
func (dac *AP235) Clear(channel int) error {
	dac.Lock()
	defer dac.Unlock()
	dac.cfg.opts._chan[C.int(channel)].DataReset = C.int(1)
	dac.sendCfgToBoard(channel)
	dac.cfg.opts._chan[C.int(channel)].DataReset = C.int(0)
	return nil
}

// Reset completely clears both data and configuration for a channel
// the error is always nil
func (dac *AP235) Reset(channel int) error {
	dac.Lock()
	defer dac.Unlock()
	dac.cfg.opts._chan[C.int(channel)].FullReset = C.int(1)
	dac.sendCfgToBoard(channel)
	dac.cfg.opts._chan[C.int(channel)].FullReset = C.int(0)
	return nil
}

// Close the dac, freeing hardware.
func (dac *AP235) Close() error {
	C.Teardown_board_corrected_buffer(dac.cfg, dac.cScatterInfo)
	errC := C.APClose(dac.cfg.nHandle)
	return enrich(errC, "APClose")
}

// Status retrieves the status of a given channel of the DAC
func (dac *AP235) Status(channel int) ChannelStatus {
	dac.Lock()
	defer dac.Unlock()
	C.rsts235(dac.cfg)
	out := ChannelStatus{Channel: channel}
	stat := dac.cfg.ChStatus[channel]
	out.FIFOEmpty = (stat>>0)&1 == 1
	out.FIFOHalfFull = (stat>>1)&1 == 1
	out.FIFOFull = (stat>>2)&1 == 1
	out.FIFOUnderflow = (stat>>3)&1 == 1
	out.BurstSingleComplete = (stat>>4)&1 == 1
	out.Busy = (stat>>5)&1 == 1
	return out
}

func (dac *AP235) doTransfer(channel int) {
	head := dac.cursor[channel]
	tailOffset := dac.sampleCount[channel] - dac.cursor[channel]
	if tailOffset > MaxXferSize {
		tailOffset = MaxXferSize
	}
	tailOffset-- // 2048 => 2047, etc.
	tail := head + tailOffset
	if tail == dac.sampleCount[channel] {
		tail--
	}
	p1 := (*C.short)(unsafe.Pointer(&dac.buffer[channel][head]))
	p2 := (*C.short)(unsafe.Pointer(&dac.buffer[channel][tail]))
	// no need for bytes to transfer, since that only applies in simple DMA mode
	// fifowro235 only writes half, we want it to write all since we are only
	// sending half to begin with
	dac.cfg.SampleCount[channel] = C.uint(tailOffset*2) + 1
	dac.cfg.current_ptr[channel] = p1
	dac.cfg.tail_ptr[channel] = p2
	C.fifowro235(dac.cfg, C.int(channel))
	dac.cursor[channel] += tailOffset + 1 // todo: wrap around
}

// CMkarrayU16 allocates a []uint16 in C and returns a Go slice without copying
// as well as the pointer for freeing, and error if malloc failed.
func cMkarrayU16(size int) ([]uint16, *C.short, error) {
	cptr := C.MkDataArray(C.int(size))
	if cptr == nil {
		return nil, nil, fmt.Errorf("cMkarrayU16: cmalloc failed")
	}
	var slc []uint16
	hdr := (*reflect.SliceHeader)(unsafe.Pointer(&slc))
	hdr.Cap = size
	hdr.Len = size
	hdr.Data = uintptr(unsafe.Pointer(cptr))
	return slc, cptr, nil
}
