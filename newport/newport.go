package newport

type motionCommand struct {
	Route       string
	Method      string
	Descr       string
	UsesAxis    bool
	DataIsArray bool
	Data        byte
	DataArray   []byte
}
