package newport

type motionCommand struct {
	Command     string
	IsRead      bool
	Descr       string
	UsesAxis    bool
	DataIsArray bool
	Data        byte
	DataArray   []byte
}

type httpSeq [2]string
