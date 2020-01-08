package sdk3

import (
	"fmt"
	"io"
	"time"

	"github.com/astrogo/fitsio"
)

func collectHeaderMetadata3(c *Camera) []fitsio.Card {
	// grab all the shit we care about from the camera so we can fill out the header
	// plow through errors, no need to bail early
	aoi, err := c.GetAOI()
	texp, err := c.GetExposureTime()
	sdkver, err := c.GetSDKVersion()
	drvver, err := c.GetDriverVersion()
	firmver, err := c.GetFirmwareVersion()
	cammodel, err := c.GetModel()
	camsn, err := c.GetSerialNumber()
	fan, err := c.GetFan()
	tsetpt, err := c.GetTemperatureSetpoint()
	tstat, err := c.GetTemperatureStatus()
	temp, err := c.GetTemperature()
	bin, err := c.GetBinning()
	binS := FormatBinning(bin)

	var metaerr string
	if err != nil {
		metaerr = err.Error()
	} else {
		metaerr = ""
	}
	now := time.Now()
	ts := fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02d",
		now.Year(),
		now.Month(),
		now.Day(),
		now.Hour(),
		now.Minute(),
		now.Second())

	return []fitsio.Card{
		/* andor-http header format includes:
		- header format tag
		- go-hcit andor version
		- sdk software version
		- driver version
		- camera firmware version

		- camera model
		- camera serial number

		- aoi top, left, top, bottom
		- binning

		- fan on/off
		- thermal setpoint
		- thermal status
		- fpa temperature
		*/
		// header to the header
		fitsio.Card{Name: "HDRVER", Value: "3", Comment: "header version"},
		fitsio.Card{Name: "WRAPVER", Value: WRAPVER, Comment: "server library code version"},
		fitsio.Card{Name: "SDKVER", Value: sdkver, Comment: "sdk version"},
		fitsio.Card{Name: "DRVVER", Value: drvver, Comment: "driver version"},
		fitsio.Card{Name: "FIRMVER", Value: firmver, Comment: "camera firmware version"},
		fitsio.Card{Name: "METAERR", Value: metaerr, Comment: "error encountered gathering metadata"},
		fitsio.Card{Name: "CAMMODL", Value: cammodel, Comment: "camera model"},
		fitsio.Card{Name: "CAMSN", Value: camsn, Comment: "camera serial number"},

		// timestamp
		fitsio.Card{Name: "DATE", Value: ts}, // timestamp is standard and does not require comment

		// exposure parameters
		fitsio.Card{Name: "EXPTIME", Value: texp.Seconds(), Comment: "exposure time, seconds"},

		// thermal parameters
		fitsio.Card{Name: "FAN", Value: fan, Comment: "on (true) or off"},
		fitsio.Card{Name: "TEMPSETP", Value: tsetpt, Comment: "Temperature setpoint"},
		fitsio.Card{Name: "TEMPSTAT", Value: tstat, Comment: "TEC status"},
		fitsio.Card{Name: "TEMPER", Value: temp, Comment: "FPA temperature (Celcius)"},
		// aoi parameters
		fitsio.Card{Name: "AOIL", Value: aoi.Left, Comment: "1-based left pixel of the AOI"},
		fitsio.Card{Name: "AOIT", Value: aoi.Top, Comment: "1-based top pixel of the AOI"},
		fitsio.Card{Name: "AOIW", Value: aoi.Width, Comment: "AOI width, px"},
		fitsio.Card{Name: "AOIH", Value: aoi.Height, Comment: "AOI height, px"},
		fitsio.Card{Name: "AOIB", Value: binS, Comment: "AOI Binning, HxV"},

		// needed for uint16 encoding
		fitsio.Card{Name: "BZERO", Value: 32768},
		fitsio.Card{Name: "BSCALE", Value: 1.0}}
}

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
