// Package nkt enables working with NKT SuperK VARIA supercontinuum laser sources.
package nkt

import (
	"time"

	"github.com/tarm/serial"
)

// MakeSerConf makes a new serial config
func MakeSerConf(addr string) *serial.Config {
	return &serial.Config{
		Name:        addr,
		Baud:        115200,
		Size:        8,
		Parity:      serial.ParityNone,
		StopBits:    serial.Stop1,
		ReadTimeout: 1 * time.Second}
}
