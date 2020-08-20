// Package imgrec contains an image recorder used to automatically save images to disk.
package imgrec

import (
	"encoding/json"
	"fmt"
	"go/types"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.jpl.nasa.gov/bdube/golab/generichttp"
)

// Recorder records image sequences with incrementing filenames in yyyy-mm-dd subfolders.  It is not thread safe.
type Recorder struct {
	// counter is the internally incrementing counter
	counter int

	// Root is the root path
	Root string

	// Prefix is the prefix for the filenames
	Prefix string

	// timeFldr is the subfolder with yyy-mm-dd format.
	timeFldr string

	// Enabled is a flag unused by this struct that allows consumers to disable its use in their code
	Enabled bool
}

// updateFolder checks the current time and updates the folder and timestamp as needed
func (r *Recorder) updateFolder() {
	now := time.Now()
	y, m, d := now.Year(), now.Month(), now.Day()
	// otherwise, timeFldr needs to be reset
	r.timeFldr = fmt.Sprintf("%04d-%02d-%02d", y, m, d)
	return
}

// mkDir makes the folder and returns it
func (r *Recorder) mkDir() (string, error) {
	fldr := path.Join(r.Root, r.timeFldr)
	err := os.MkdirAll(fldr, 0777)
	return fldr, err
}

// Write implements io.Writer and writes the contents of a fits file to disk
func (r *Recorder) Write(p []byte) (n int, err error) {
	// make sure the folder exists
	r.updateFolder()
	fldr, err := r.mkDir()
	if err != nil {
		return 0, err
	}

	// now open the file and write to it
	fn := fmt.Sprintf("%s%06d.fits", r.Prefix, r.counter)
	fn = path.Join(fldr, fn)
	var fid *os.File
	fid, err = os.OpenFile(fn, os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil && os.IsNotExist(err) {
		fid, err = os.Create(fn)
		if err != nil {
			return 0, err
		}
	}
	defer fid.Close()
	if err != nil {
		return 0, err
	}
	return fid.Write(p)
}

// Incr updates the filename counter; it scans the folder to do so.  If there is an error, the counter is not incremented
func (r *Recorder) Incr() {
	dn, _ := r.mkDir()
	files, err := ioutil.ReadDir(dn)
	if err != nil {
		return
	}
	count := 0
	for _, file := range files {
		// skip directories, non-fits, and wrong prefix
		if file.IsDir() {
			continue
		}
		fn := file.Name()
		if !strings.HasSuffix(fn, ".fits") || !strings.HasPrefix(fn, r.Prefix) {
			continue
		}
		// guaranteed match
		bit := strings.Split(fn, r.Prefix)[1]
		bit = bit[:len(bit)-5] // pop fits
		n, err := strconv.Atoi(bit)
		if err != nil {
			return
		}
		if count < n {
			count = n
		}
	}
	r.counter = count + 1
}

// HTTPWrapper is an HTTP wrapper around an image recorder that allows the folder and prefix to be changed on the fly
//
// it does not implement generichttp.HTTPer, offering an Inject method allowing it to be injected
// into another HTTPer
type HTTPWrapper struct {
	*Recorder
}

// NewHTTPWrapper returns an HTTP wrapper around a recorder
func NewHTTPWrapper(r *Recorder) HTTPWrapper {
	return HTTPWrapper{r}
}

// SetRoot updates the root folder of the recorder
func (h HTTPWrapper) SetRoot(w http.ResponseWriter, r *http.Request) {
	str := generichttp.StrT{}
	err := json.NewDecoder(r.Body).Decode(&str)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rec := h.Recorder
	rec.Root = str.Str
	rec.updateFolder()
	_, err = rec.mkDir()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// GetRoot gets the recorder's root folder and sends it back as JSON
func (h HTTPWrapper) GetRoot(w http.ResponseWriter, r *http.Request) {
	hp := generichttp.HumanPayload{T: types.String, String: h.Recorder.Root}
	hp.EncodeAndRespond(w, r)
}

// SetPrefix updates the filename prefix of the recorder
func (h HTTPWrapper) SetPrefix(w http.ResponseWriter, r *http.Request) {
	str := generichttp.StrT{}
	err := json.NewDecoder(r.Body).Decode(&str)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.Recorder.Prefix = str.Str
	h.Recorder.counter = 0
	w.WriteHeader(http.StatusOK)
}

// GetPrefix gets the recorder's prefix and sends it back as JSON
func (h HTTPWrapper) GetPrefix(w http.ResponseWriter, r *http.Request) {
	hp := generichttp.HumanPayload{T: types.String, String: h.Recorder.Prefix}
	hp.EncodeAndRespond(w, r)
}

// GetEnabled returns the Recorder's Enabled field
func (h HTTPWrapper) GetEnabled(w http.ResponseWriter, r *http.Request) {
	hp := generichttp.HumanPayload{T: types.Bool, Bool: h.Recorder.Enabled}
	hp.EncodeAndRespond(w, r)
	return
}

// SetEnabled sets the recorder's Enabled field
func (h HTTPWrapper) SetEnabled(w http.ResponseWriter, r *http.Request) {
	bT := generichttp.BoolT{}
	err := json.NewDecoder(r.Body).Decode(&bT)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.Recorder.Enabled = bT.Bool
	return
}

// Inject adds GET and POST routes for /autorwrite/root and /autowrite/prefix to the HTTPer which manipulate this wrapper's recorder
func (h HTTPWrapper) Inject(other generichttp.HTTPer) {
	rt := other.RT()
	rt[generichttp.MethodPath{Method: http.MethodPost, Path: "/autowrite/root"}] = h.SetRoot
	rt[generichttp.MethodPath{Method: http.MethodGet, Path: "/autowrite/root"}] = h.GetRoot
	rt[generichttp.MethodPath{Method: http.MethodPost, Path: "/autowrite/prefix"}] = h.SetPrefix
	rt[generichttp.MethodPath{Method: http.MethodGet, Path: "/autowrite/prefix"}] = h.GetPrefix
	rt[generichttp.MethodPath{Method: http.MethodPost, Path: "/autowrite/enabled"}] = h.SetEnabled
	rt[generichttp.MethodPath{Method: http.MethodGet, Path: "/autowrite/enabled"}] = h.GetEnabled
}
