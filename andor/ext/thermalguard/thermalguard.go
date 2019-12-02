/*Package thermalguard extends the andor package to provide a continuously running
thermal guardian that can be signaled to save the camera.  This is used in, for
example, scenarios where power is lost but a UPS provides short-term
continuance.  The guardian will walk the camera thermal set point
back to 20C at a rate of 5C/min and then shut down the camera after this
procedure has completed.

To use it, simply replicate this example:

 import tg "thermalguard"

 // the creation of the guaridan just fills in a struct
 cam := andor.Camera{...} // prepare your camera
 royalMilitia := tg.Guardian{cam}

 // after being made aware of dangerous times ahead:
 // you might want to do this concurrently so you can save other bacon at the same time
 go royalMilitia.SaveMe()
 // This sets off:
 // T0:    -100C
 // T+1m:  -95C
 // T+2m:  -90C
 // ...
 // T+20m: -0C
 // T+24m: 20C
 // T+25m: cam.Shutdown() is invoked
*/
package thermalguard

import (
	"github.jpl.nasa.gov/HCIT/go-hcit/andor"
)

const tempStep = 5 // Celcius
type Guardian stuct {
	cam *andor.Camera
}

// Save me walks the temperature setpoint to 20C at a rate of 5C per minute
func (g *Guardian) SaveMe() {
	ticker := time.NewTicker(1 * time.Minute)
	done := make(chan bool)

	temperature := g.cam.GetTemperature()
	numTicksNeeded = int(math.Ceil((20 - temperature) / tempStep))
	ticks := 0
	for {
		select {
		case <- done:
			return
		case <- ticker.C:
			temperature += tempStep
			g.cam.SetTemperature(temperature)
			if ticks == numTicksNeeded { done <- true }
			ticks++
		}
	}
}