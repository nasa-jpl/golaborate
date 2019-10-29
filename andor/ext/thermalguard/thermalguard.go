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
royalMilitia := tg.NewGuardian(cam)

// after being made aware of dangerous times ahead:
// you might want to do this concurrently so you can save other bacon at the same time
go royalMilitia.SaveMe()
// Example:
// T0:    -100C
// T+1m:  -95C
// T+2m:  -90C
// ...
// T+20m: -0C
// T+24m: 20C
// T+25m: cam.Shutdown() is invoked
*/
package thermalguard
