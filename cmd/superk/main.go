package main

import (
	"fmt"

	"github.jpl.nasa.gov/HCIT/go-hcit/nkt"
)

func main() {
	// 	payload := uint16(5000)
	// 	payload_b := make([]byte, 2)
	// 	binary.LittleEndian.PutUint16(payload_b, payload)

	// this case is "Example 1" in the manual and encodes properly
	// msg := nkt.MakeTelegram(0x0F, 0xA2, 0x05, 0x30, []byte{0x03})

	// this case is 'example 2" in the manual
	// msg := nkt.MakeTelegram(0x0A, 0xA2, nkt.MessageTypes["Write"], 0x23, payload_b)
	// msg, _ := nkt.DecodeTelegram([]byte{0x0D, 0xA2, 0x0F, 0x03, 0x30, 0x48, 0x2F, 0x0A})

	// this tests getting the typecode for a single fire
	// m := nkt.NewSuperKExtreme("192.168.100.187:2116")
	// mp, hp, err := m.GetValue("TypeCode")
	// fmt.Printf("%+v\n", mp)
	// fmt.Printf("%+v\n", hp)
	// fmt.Println(err)

	// this does it ten times in a row at some sort of reprate
	// result: 20 ms repreate is the knife's edge for the NKT rejecting some connections
	// vals := util.ArangeByte(10)
	// for range vals {
	// 	mp, hp, err := m.GetValue("TypeCode")
	// 	fmt.Printf("%+v\n", mp)
	// 	fmt.Printf("%+v\n", hp)
	// 	fmt.Println(err)
	// 	time.Sleep(20 * time.Millisecond)
	// }

	// in this case we make a telegram that replicates m.GetValue("TypeCode") "in the raw" to verify if AddressScan
	// is the problem
	// conn, err := util.TCPSetup("192.168.100.187:2116", 3*time.Second)
	// if err != nil {
	// 	panic(err)
	// }
	// reader := bufio.NewReader(conn)
	// tele, _ := nkt.MakeTelegram(nkt.MessagePrimitive{
	// 	Dest:     0x0F,
	// 	Src:      0xA1,
	// 	Register: 0x61,
	// 	Type:     "Read",
	// })

	// for range util.ArangeByte(10) {
	// 	_, err := conn.Write(tele)
	// 	if err != nil {
	// 		fmt.Println("L60 ", err)
	// 	}

	// 	// return bufio.NewReader(conn).ReadBytes(telEnd)
	// 	buf, err := reader.ReadBytes(0x0A)
	// 	if err != nil {
	// 		fmt.Println("L66 ", err)
	// 	}
	// 	fmt.Println("L68 ", buf)
	// }

	// conn.Close()
	m, err := nkt.AddressScan("192.168.100.187:2116")
	fmt.Println(m)
	fmt.Println(err)
}
