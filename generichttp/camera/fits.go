package camera

import (
	"image"
	"io"
	"reflect"
	"unsafe"

	"github.com/astrogo/fitsio"
)

// WriteFits streams a fits file to w
func WriteFits(w io.Writer, metadata []fitsio.Card, imgs []image.Image) error {
	metadata = append(metadata, fitsio.Card{Name: "BZERO", Value: 32768}, fitsio.Card{Name: "BSCALE", Value: 1.0})
	nframes := len(imgs)
	b := imgs[0].Bounds()
	width, height := b.Dx(), b.Dy()
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

	bufSize := 1
	for _, s := range dims {
		bufSize *= s
	}
	ints := make([]int16, bufSize)
	offset := 0
	for _, img := range imgs {
		imgConcrete := (img).(*image.Gray16)
		uints := bytesToUint(imgConcrete.Pix)
		l := len(uints)
		for idx := 0; idx < l; idx++ {
			ints[offset+idx] = int16(uints[idx] - 32768)
		}
		offset += l
	}
	err = im.Write(ints)
	if err != nil {
		return err
	}
	return fits.Write(im)
}

func bytesToUint(b []byte) []uint16 {
	var ary []uint16
	hdr := (*reflect.SliceHeader)(unsafe.Pointer(&ary))
	hdr.Data = uintptr(unsafe.Pointer(&b[0]))
	hdr.Len = len(b) / 2
	hdr.Cap = cap(b) / 2
	return ary
}
