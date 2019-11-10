/*Package camera describes a standard set of interfaces for control of cameras

The Minimal type contains the basics, while Sci contains some extended features
typically found on scientific cameras.

*/
package camera

// Minimal describes a minimal camera interface with only the basics.
type Minimal interface {
	// Initialize initializes the camera.  This may have myriad side effects,
	// for example the initialization of a camera driver in C,
	// the allocation of buffer(s) for holding camera frames,
	// and the setting of hardware parameters like shift speeds
	// on CCD cameras, or activation of cooling and adjustment
	// of temperature setpoint, etc.
	Initialize() error

	// Finalize finalizes the camera, which may have myriad side effects
	// but most prominently, will typically call a similar function
	// on the camera driver
	Finalize() error

	// GetRes gets the (H, W) associated with the data returned by GetFrameXX
	GetRes() ([2]int, error)

	// GetFrameU16 gets a frame as uint16.  The data is a 1D slice which is
	// strided by the frame height.
	GetFrameU16() (*[]uint16, error)

	// GetFrameI32 gets a frame as int32.  The data is a 1D slice which is
	// strided by the frame height.
	GetFrameI32() (*[]int32, error)
}

// Sci describes an extended interface for scientific cameras
// we do not enforce this constraint, but a type which implements
// Sci will nearly always implement Minimal.
type Sci interface {
	// GetTempControlActive gets if internal temperature control
	// is currently running
	GetTempControlActive() (bool, error)

	// SetTempControlActive turns internal temperature control on or off
	SetTempControlActive(bool) error

	// GetTempSetpoint gets the temperature setpoint in Celcius
	GetTempSetpoint() (float64, error)

	// SetTempSetpoint sets the temperature setpoint in Celcius
	SetTempSetpoint(float64) error

	// GetTemp gets the current camera temperature in Celcius
	// what the temperature is actually measured on (sensor, pcb, etc)
	// is implementation dependent.
	GetTemp() (float64, error)
}
