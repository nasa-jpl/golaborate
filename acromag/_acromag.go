/* Package acromag provides a standardized interface to Acromag AcroPack DAC modules.

Currently, only the AP236 isolated 16-bit module and AP235 16-bit waveform module
are supported.  The AP236 is not recommended for future projects, as it is noisier
than the AP235.  The AP231 should be used instead, and a wrapper made in this pkg.

The DAC modules have a common interface.  Sample usage looks like:

 dac, err := acromag.NewAP236(0)
 // do with err
 // configuration
 channels := []int{0,1,2}
 for _, ch := range channels {
		err = dac.SetClearVoltage(ch, ap235.MidScale)
		if err != nil {
			return dac, err
		}
		err = dac.SetPowerUpVoltage(ch, ap235.MidScale)
		if err != nil {
			return dac, err
		}
		err = dac.SetRange(ch, "-10,10")
		if err != nil {
			return dac, err
		}
		err = dac.SetOverRange(ch, false)
		if err != nil {
			return dac, err
		}

		err = dac.SetOutputSimultaneous(ch, false)
		if err != nil {
			return dac, err
		}

		// this means output glitches if the FIFO is emptied
		// instead of playback stopping
		err = dac.SetClearOnUnderflow(ch, false)
		if err != nil {
			return dac, err
		}

		// lastly, power up the DAC channel
		err = dac.Output(ch, 0)
		if err != nil {
			return dac, err
		}
	}

	// usage
	dac.Output(0, 3.3) # 3.3V on channel 0
	dac.OutputDN(0, 30000) # 30,000 DN
	dac.OutputMulti(channels, []float64{0,1,2}) # put 0V on ch0, 1V on ch1, ...

	// note: outputMulti does simultaneous triggering

*/
package acromag

/*
#cgo LDFLAGS: -lm
#include <stdlib.h>
#include "apcommon.h"
*/
import "C"
import "fmt"

func init() {
	errCode := C.InitAPLib()
	if errCode != C.S_OK {
		panicS := fmt.Sprintf("initializing Acromag library failed with code %d", errCode)
		panic(panicS)
	}
}
