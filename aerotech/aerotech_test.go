package aerotech

import (
	"math/rand"
	"testing"
)

func Benchmark32BitBitfieldUnpackManualInline(b *testing.B) {
	var (
		bf = rand.Int31()
		s  Status
	)
	for i := 0; i < b.N; i++ {
		s.Enabled = (bf>>0)&1 == 1
		s.Homed = (bf>>1)&1 == 1
		s.InPosition = (bf>>2)&1 == 1
		s.MoveActive = (bf>>3)&1 == 1
		s.AccelPhase = (bf>>4)&1 == 1
		s.DecelPhase = (bf>>5)&1 == 1
		s.PositionCapture = (bf>>6)&1 == 1
		s.CurrentClamp = (bf>>7)&1 == 1
		s.BrakeOutput = (bf>>8)&1 == 1
		s.MotionIsCw = (bf>>9)&1 == 1
		s.MasterSlaveControl = (bf>>10)&1 == 1
		s.CalActive = (bf>>11)&1 == 1
		s.CalEnabled = (bf>>12)&1 == 1
		s.JoystickControl = (bf>>13)&1 == 1
		s.Homing = (bf>>14)&1 == 1
		s.MasterSuppress = (bf>>15)&1 == 1
		s.GantryActive = (bf>>16)&1 == 1
		s.GantryMaster = (bf>>17)&1 == 1
		s.AutofocusActive = (bf>>18)&1 == 1
		s.CommandFilterDone = (bf>>19)&1 == 1
		s.InPosition2 = (bf>>20)&1 == 1
		s.ServoControl = (bf>>21)&1 == 1
		s.CwEOTLimit = (bf>>22)&1 == 1
		s.CcwEOTLimit = (bf>>23)&1 == 1
		s.HomeLimit = (bf>>24)&1 == 1
		s.MarkerInput = (bf>>25)&1 == 1
		s.HallAInput = (bf>>26)&1 == 1
		s.HallBInput = (bf>>27)&1 == 1
		s.HallCInput = (bf>>28)&1 == 1
		s.SineEncoderError = (bf>>29)&1 == 1
		s.CosineEncoderError = (bf>>30)&1 == 1
		s.ESTOPInput = (bf>>31)&1 == 1
	}
	_ = s
}

func Benchmark32BitBitfieldUnpackFuncCall(b *testing.B) {
	var (
		bf = rand.Int31()
		s  Status
	)
	for i := 0; i < b.N; i++ {
		s = StatusFromBitfield(bf)
	}
	_ = s // prevent unused var
}
