// this file contains a few small image processing utilities
package camera

// in-place reverse of a slice
func reverse(slice []uint16) []uint16 {
	l := len(slice)
	for i := l/2 - 1; i != 0; i-- {
		opp := l - 1 - i
		slice[i], slice[opp] = slice[opp], slice[i]
	}
	return slice
}

// ImRot90 rotates an image 90 degrees clockwise
func ImRot90(stridedBuffer []uint16, width, height int) []uint16 {
	out := make([]uint16, len(stridedBuffer))
	// the data is row major, we're just swapping to column major.
	// so, we iterate the row index and use it to grab columns
	src := reverse(stridedBuffer)
	top := width * height
	for idx := 0; idx < top; idx++ {
		i := idx / width
		j := idx % width
		out[idx] = stridedBuffer[height*j+i]
	}
	return out
}

// ImRot180 rotates an image 180 degrees clockwise
func ImRot180(stridedBuffer []uint16, width, height int) []uint16 {
	return stridedBuffer
}

// ImRot270 rotates an image 270 degrees clockwise
func ImRot270(stridedBuffer []uint16, width, height int) []uint16 {
	return stridedBuffer
}
