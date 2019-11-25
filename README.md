# go-hcit

This monorepo contains a number of packages written predominantly in the Go programming language for interacting with lab hardware -- sensors, motion controllers, cameras, and deformable mirrors.  The HTTP server extensions to these packages add less than 1 ms of latency to the communication and enable a more pleasant API and thinner client libraries in any language.  The Andor server uses CGo and is less portable for that reason.  See README in the andor directory for special compilation instructions.

It also includes some lower level libraries for allowing transparent use of serial or TCP connections to these devices and connection keep-alive behavior on either connection type, as well as graceful backoff in the event of devices rejecting connection thrashing.

The following hardware is supported:

DMs:
- JPL Gen 5 DM controllers
- BMC commercial DM controllers

Motion:
- Newport ESP 300 and 301 motion controllers
- Newport XPS motion controllers
- Aerotech Epaq and Ensemble motion controllers

Light sources:
- NKT SuperK supercontinuum lasers
- IXL Lightwave laser diodes

Environmental Sensors and controllers:
- Granville-Phillips GP375 Convectron pressure sensors
- Lesker KJC 300 pressure sensors
- Fluke DewK 1620a temperature and humidity sensors
- Lakeshore 322 thermometers and temperature controllers
- Cryo-Con temperature monitors

Instruments:
- Zygo interferometers


## Setup

Most of these servers are written in golang.  Due to our need to compile for windows XP, we are using Golang 1.9.  Hopefully soon we will be able to updated to 1.13 or the latest with the replacement of the Zygo PC running windows XP.

To install golang, grab the binaries from http://golang.org/dl, use your system package manager, or [brew](https://brew.sh/) on MacOS.  For brew:

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
go get github.com/cenkalti/backoff  # graceful backoff when connections rejected by hardware

go get github.com/astrogo/fitsio  # andor binaries need fits, no need for the rest
go get github.com/lordadamson/cgo.wchar  # if you intend to build the Andor/sdk3 pkg
```

There are no external dependencies aside from these .

If you need to modify a program, cd from `go-hcit` to `/cmd/<the program>` and edit `main.go`.  Then `cd ..` and run:

```sh
make <command>
```

For example,

```sh
make superk
```
`GOOS`, "go operating system" should be appropriate for the machine you intend to run the software on.  `GOARCH` is the processor architecture, which should generally be 386 (32-bit) or amd64 (64-bit).  The complete list of acceptable values for these constants can be found at https://golang.org/doc/install/source#environment

Note that go supports cross compilation, so compiling for linux or windows from a mac is a nonissue (excluding the andor module).


## Binary servers

see the `cmd` directory for the top-level source code of the server binaries.  When this project reaches maturity,
a collection of the binaries for various architectures and platforms will be stored on the S383 netowrk.  For now, build
a binary to use it.  The `superk` program is running continuously on misery.

## Documentation

To view the documentation for the `Go` code, cd to the root of this repository under `$GOPATH`, then run `godoc -http=:6060` and visit http://localhost:6060/pkg/github.jpl.nasa.gov/HCIT/go-hcit/ in your browser.

To view the documentation for the HTTP endpoints, the current best approach is to clone [swagger-ui](https://github.com/swagger-api/swagger-ui), then `npm start` to run the serve, and paste `docs/http-documentation.yaml` into the editor.

## development status

| Device            | driver | server |
| :---              | :----: |  ---:  |
| JPL DM Controller |        |        |
| BMC commercial    | X      |  X     |
| Andor cameras     | ~X     |        |
| other cameras (?) |        |        |
| Newport EPS       |  X     |  X     |
| Newport XPS       |        |        |
| Aerotech Ensemble |        |        |
| PI motion         |        |        |
| Lakeshore temp    |  X     |        |
| Fluke temp/RH     |  X     |  X     |
| Lesker pressure   |  X     |  X     |
| GP pressure       |  X     |  X     |
| Cryo-Con thermal  |  X     |  X     |
| Omega flowmeter   |        |        |
| Omega temp        |        |        |
| APC UPS           |        |        |
| Pfeiffer turbo    |        |        |
| NKT               |  X     |  X     |
| IXL diode         |  X     |  X     |
| Thermocube chiller|        |        |
| Zygo              |  X     | X      |
| 4D (?)            |        |        |
