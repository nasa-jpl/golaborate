package pi

type Enabler interface {
	// Enable enables an axis
	Enable(string) error

	// Disable disables an axis
	Disable(string) error

	// GetEnabled gets if an axis is enabled
	GetEnabled(string) (bool, error)
}

type InPositionQueryer interface {
	// GetInPosition returns True if the axis is in position
	GetInPosition(string) (bool, error)
}

type Mover interface {
	// GetPos gets the current position of an axis
	GetPos(string) (float64, error)

	// MoveAbs moves an axis to an absolute position
	MoveAbs(string, float64) error

	// MoveRel moves an axis a relative amount
	MoveRel(string, float64) error

	// Home homes an axis
	Home(string) error
}

type Speeder interface {
	// SetVelocity sets the velocity setpoint on the axis
	SetVelocity(string, float64) error

	// GetVelocity gets the velocity setpoint on the axis
	GetVelocity(string) (float64, error)
}

type RawCommunicator interface {
	Raw(string) (string, error)
}
