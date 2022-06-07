package nkt

import (
	"errors"
	"sync"
	"time"

	"github.jpl.nasa.gov/bdube/golab/generichttp/laser"
)

type MockSuperK struct {
	sync.Mutex
	emissionruntime uint32
	variaLo         uint16
	variaHi         uint16
	power           uint16
	emission        bool
	cancel          chan struct{}
}

func NewMockSuperK(addr string, connectSerial bool) *MockSuperK {
	return &MockSuperK{
		cancel: make(chan struct{}),
	}
}

func (m *MockSuperK) runtimeTicker() {
	t := time.NewTicker(time.Second)
	for {
		select {
		case <-t.C:
			m.Lock()
			m.emissionruntime++
			m.Unlock()
		case <-m.cancel:
			t.Stop()
			break
		}
	}
}
func (m *MockSuperK) SetEmission(b bool) error {
	m.Lock()
	defer m.Unlock()
	// if we are already emitting and the user wants us to emit, do nothing
	if m.emission && b {
		return nil
	}
	// if we are not emitting and they do not want us to emit, do nothing
	if !m.emission && !b {
		return nil
	}
	// if we are already emitting and they do not want us to emit, stop
	if m.emission && !b {
		// stop emission
		m.cancel <- struct{}{}
		m.emission = false
	}
	// if we are not emitting and they want us to emit, begin emitting
	if !m.emission && b {
		m.emission = true
		go m.runtimeTicker()
	}
	return nil
}

func (m *MockSuperK) GetEmission() (bool, error) {
	m.Lock()
	defer m.Unlock()
	return m.emission, nil
}

func (m *MockSuperK) SetPower(p float64) error {
	m.Lock()
	defer m.Unlock()
	if p > 100 || p < 0 {
		return errors.New("nkt: commanded power was outside the range [0,100]")
	}
	m.power = uint16(p * 10)
	return nil
}

func (m *MockSuperK) GetPower() (float64, error) {
	m.Lock()
	defer m.Unlock()
	return float64(m.power / 10), nil
}

func (m *MockSuperK) SetShortWave(nanometers float64) error {
	m.Lock()
	defer m.Unlock()
	m.variaLo = uint16(nanometers * 10)
	return nil
}

func (m *MockSuperK) SetLongWave(nanometers float64) error {
	m.Lock()
	defer m.Unlock()
	m.variaHi = uint16(nanometers * 10)
	return nil
}

func (m *MockSuperK) GetShortWave() (float64, error) {
	m.Lock()
	defer m.Unlock()
	return float64(m.variaLo / 10), nil
}

func (m *MockSuperK) GetLongWave() (float64, error) {
	m.Lock()
	defer m.Unlock()
	return float64(m.variaHi / 10), nil
}

func (m *MockSuperK) SetCenterBandwidth(cbw laser.CenterBandwidth) error {
	short, long := cbw.ToShortLong()
	err := m.SetShortWave(short)
	if err != nil {
		return err
	}
	err = m.SetLongWave(long)
	if err != nil {
		return err
	}
	return nil
}

func (m *MockSuperK) GetCenterBandwidth() (laser.CenterBandwidth, error) {
	var ret laser.CenterBandwidth
	short, err := m.GetShortWave()
	if err != nil {
		return ret, err
	}
	long, err := m.GetLongWave()
	if err != nil {
		return ret, err
	}
	ret = laser.ShortLongToCB(short, long)
	return ret, err
}

func (m *MockSuperK) StatusMain() (map[string]bool, error) {
	em, err := m.GetEmission()
	if err != nil {
		return nil, err
	}
	return map[string]bool{
		"Emission on":                        em,
		"Interlock signal off":               false,
		"Interlock loop input low":           false,
		"Interlock loop output low":          false,
		"Module disabled":                    false,
		"Supply voltage out of range":        false,
		"Board temperature out of range":     false,
		"Heat sink temperature out of range": false,
	}, nil
}

func (m *MockSuperK) StatusVaria() (map[string]bool, error) {
	return map[string]bool{
		"Interlock off":      false,
		"Interlock loop in":  false,
		"Interlock loop out": false,
		"Supply voltage low": false,
		"Shutter sensor 1":   false,
		"Shutter sensor 2":   false,
		"Filter 1 moving":    false,
		"Filter 2 moving":    false,
		"Filter 3 moving":    false,
		"Error code present": false,
	}, nil
}
