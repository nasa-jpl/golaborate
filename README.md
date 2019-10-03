# go-hcit
 golang servers/services for S383's high contrast imaging testbeds.  This is set up as a monorepo and contains several packages.  Below is a somewhat infrequently maintained index of the packages and what they enable.  Each type of sensor has a `ReadAndReplyWithJSON` method which implements `http.Handler`

 ### commonpressure

 refactored, common logic for working with pressure sensors.

 ### fluke

 Reading from Fluke 1620a "DewK" temp/humidity sensors over TCP/IP or serial.

 ### granville-phillips

 Reading from GP375 pressure meters over serial.

 ### lakeshore

 Reading from a 332 sensor/heater controller.

 ### lesker

 Reading from KJC300 pressure meters.

 ### thermocube

 Reading from Thermocube 200~400 series chillers.

 In /cmd, there is the source for several executables:

 ### envsrv

 This server has routes for each sensor on OMC/GPCT/DST and allows them to be queried via HTTP.

 ### zygo

 This service enables remote measurement with Zygo interferometers via HTTP.


## Setup

Most of these servers are written in golang.  Due to our need to compile for windows XP, we are using Golang 1.9.  Hopefully soon we will be able to updated to 1.13 or the latest with the replacement of the Zygo PC running windows XP.

To install golang, grab the binaries from http://golang.org/dl, use your system package manager, or brew on macos.  For brew:

```
brew install go@1.9
export $GOPATH=$HOME/go
```

Golang has the convention that all of your code has to be on the `$GOPATH` under `/src`.  That code has to be in the appropriate folder.  To set this up from your shell, do the following:

```sh
cd ~/go/src
mkdir github.jpl.nasa.gov && cd github.jpl.nasa.gov
mkdir HCIT && cd HCIT
git clone https://github.jpl.nasa.gov/HCIT/go-hcit

go get github.com/tarm/serial  # talking to serial devices
go get github.com/spf13/viper  # configuration
go get gopkg.in/yaml.v2        # YAML file support for configs
go get github.com/snksoft/crc  # Cyclic Redundancy Check library for NKT devices
```

There are no external dependencies aside from these .

If you need to modify a program, cd from `go-hcit` to `/cmd/<the program>` and edit `main.go`.  Then run:

```sh
env GOOS=linux GOARCH=386 go build main.go
```

`GOOS`, "go operating system" should be appropriate for the machine you intend to run the software on.  `GOARCH` is the processor architecture, which should generally be 386 (32-bit) or amd64 (64-bit).  The complete list of acceptable values for these constants can be found at https://golang.org/doc/install/source#environment

Note that go supports cross compilation, so compiling for linux or windows from a mac is a nonissue.


## Writing a new device "driver"

Most pieces of lab equipment communicate with a computer over RS232, aka a Serial interface.  In Golang, a profitable way to communicate with them is to connect with `tarm/serial`, an excellent library for raw serial functions, and use `bufio.Reader.ReadBytes` to scan the reply, even if it comes in multiple replies, up to a termination byte(s).  To make a "driver" for any new hardware, you can simply consult its manual for the following information:

- baudrate
- number of data bits
- number of stop bits
- parity
- message terminations

Then copy paste one of the existing modules, replace this information, and update the gofuncs as appropriate for the hardware controls or queries you want to implement.
