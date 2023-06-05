package thermotek

import "testing"

func TestChecksumManualExample(t *testing.T) {
	msg := []byte{'.', '0', '1', '0', '1', 'W', 'a', 't', 'c', 'h', 'D', 'o', 'g'}
	cs := checksum(msg)
	if cs[0] != '0' || cs[1] != '1' {
		t.Fatalf("expected checksum to be 01, got %x", cs)
	}
}

func TestWholeMessageManualExample(t *testing.T) {
	msg := []byte{'0', '1', 'W', 'a', 't', 'c', 'h', 'D', 'o', 'g'}
	msg2 := frameMessage(msg)
	truth := []byte{'.', '0', '1', '0', '1', 'W', 'a', 't', 'c', 'h', 'D', 'o', 'g', '0', '1', '\r'}
	if len(msg2) != len(truth) {
		t.Fatal("encoded message and truth from manual differ in length")
	}
	for i := 0; i < len(msg2); i++ {
		if msg2[i] != truth[i] {
			t.Errorf("byte %d mismatch, expected %c got %c", i, truth[i], msg2[i])
		}
	}
}

func TestResponseUnframing(t *testing.T) {
	resp := []byte{'#', '0', '1', '0', '1', '0', 'W', 'a', 't', 'c', 'h', 'D', 'o', 'g', '0', '1', '0', '0', 'E', '7', '\r'}
	data, err := checkAndUnframeResponse(resp)
	if err != nil {
		t.Log(err.Error())
		t.FailNow()
	}
	truthData := []byte{'0', '1', '0', '0'}
	if len(data) != len(truthData) {
		t.Fatal("unframed response and true data differ in length")
	}
	for i := 0; i < len(data); i++ {
		if data[i] != truthData[i] {
			t.Errorf("byte %d mismatch, expected %c got %c", i, truthData[i], data[i])
		}
	}
}
