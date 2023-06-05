package thermotek

const (
	hextable = "0123456789ABCDEF"
)

func hexEncodeByte(b byte) [2]byte {
	return [2]byte{
		hextable[b>>4],
		hextable[b&0x0f],
	}
}
