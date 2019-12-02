package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"goji.io"
	"goji.io/pat"

	"github.jpl.nasa.gov/HCIT/go-hcit/andor/sdk3"

	"net/http/pprof"
	_ "net/http/pprof"
)

var (
	// Version is the version number.  Typically injected via ldflags with git build
	Version = "dev"
)

func main() {
	var cmd string
	args := os.Args
	if len(args) == 1 {
		// no args given, print help
	} else {
		cmd = args[1]
		fmt.Println(cmd)
		return
	}
	err := sdk3.InitializeLibrary()
	if err != nil {
		return

	}
	defer sdk3.FinalizeLibrary()
	ncam, err := sdk3.DeviceCount()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%d cameras\n", ncam)
	swver, err := sdk3.SoftwareVersion()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("version %s\n", swver)

	var idx int
	if len(os.Args) > 1 {
		idx, _ = strconv.Atoi(os.Args[1])
	} else {
		idx = 0
	}

	c, err := sdk3.Open(idx)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer c.Close()
	err = sdk3.JPLNeoBootup(c)
	fmt.Println(err)

	fmt.Println(sdk3.GetString(c.Handle, "SerialNumber"))

	c.Allocate()
	err = c.QueueBuffer()
	fmt.Println("queue error", err)

	w := sdk3.NewHTTPWrapper(c)
	mux := goji.NewMux()
	w.RT().Bind(mux)
	mux.HandleFunc(pat.New("/debug/pprof/"), pprof.Index)
	mux.HandleFunc(pat.New("/debug/pprof/cmdline"), pprof.Cmdline)
	mux.HandleFunc(pat.New("/debug/pprof/profile"), pprof.Profile)
	mux.HandleFunc(pat.New("/debug/pprof/symbol"), pprof.Symbol)
	mux.HandleFunc(pat.New("/debug/pprof/trace"), pprof.Trace)
	log.Fatal(http.ListenAndServe(":8000", mux))
}
