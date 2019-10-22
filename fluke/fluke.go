// Package fluke enables working with 1620a DewK Temp/Humidity sensors
package fluke

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// TempHumid holds Temperature and Humidity data, T in C and H in % RH
type TempHumid struct {
	T float64 `json:"temp"`
	H float64 `json:"rh"`
}

// EncodeAndRespond Encodes the data to JSON and writes to w.
// logs errors and replies with http.Error // status 500 on error
func (th *TempHumid) EncodeAndRespond(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(th)
	if err != nil {
		fstr := fmt.Sprintf("error encoding fluke data to json state %q", err)
		log.Println(fstr)
		http.Error(w, fstr, http.StatusInternalServerError)
	}
	return
}

// ParseTHFromBuffer converts a raw buffer looking like 21.4,6.5,0,0\r to a TempHumid object for channel 1
func ParseTHFromBuffer(buf []byte) (TempHumid, error) {
	// convert it to a string, then split on ",", and cast to float64
	str := string(buf)
	pieces := strings.SplitN(str, ",", 3)[:2] // 3 pieces potentially leaves the trailing trash, [:2] drops it
	numeric := make([]float64, 2)
	for i, v := range pieces {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return TempHumid{}, err
		}
		numeric[i] = f
	}
	return TempHumid{T: numeric[0], H: numeric[1]}, nil
}
