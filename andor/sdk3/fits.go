package sdk3

import (
	"io"

	"github.com/astrogo/fitsio"
)

// writeFits streams a fits file to w
func writeFits(w io.Writer, metadata []fitsio.Card, buffer []uint16, width, height, nframes int) error {
	fits, err := fitsio.Create(w)
	if err != nil {
		return err
	}
	defer fits.Close()
	dims := []int{width, height}
	if nframes > 1 {
		dims = append(dims, nframes)
	}
	im := fitsio.NewImage(16, dims)
	defer im.Close()
	err = im.Header().Append(metadata...)
	if err != nil {
		return err
	}

	// investigated on the playground, this can't be done with slice dtype hacking
	// https://play.golang.org/p/HvR74t5sbbd
	// so the alloc and underflow is necessary, unfortunate since for a big cube it could mean a multi-GB alloc
	bufOut := make([]int16, len(buffer))
	for idx := 0; idx < len(buffer); idx++ {
		bufOut[idx] = int16(buffer[idx] - 32768)
	}
	err = im.Write(bufOut)
	if err != nil {
		return err
	}
	return fits.Write(im)
}
