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

Cameras:
- Andor Neo sCMOS and iXON EMCCD (full SDK for both)

## Installation and Compilation

Most of these servers are written in Go.  Go has a statement of absolute compatability within the 1.x version series, so you may use any version that is recent enough for the dependencies.  At the time of writing, this is Go 1.12.  If you need to compile for XP (e.g., for a MetroPro remote server) use Go 1.9.  The dependency tree for `cmd/zygoserver` is compatible with Go 1.9 and includes several backports from newer versions of the language to facilitate this.

To install golang, grab the binaries from http://golang.org/dl, use your system package manager, or [brew](https://brew.sh/) on MacOS.  For brew:

```
brew install go  # you can use go@1.9 to get v1.9
export $GOPATH=$HOME/go
```

Golang has the convention that all of your code has to be on the `$GOPATH` under `/src`.  That code has to be in the appropriate folder.  To set this up from your shell, do the following:

```sh
cd ~/go/src
mkdir github.jpl.nasa.gov && cd github.jpl.nasa.gov
mkdir HCIT && cd HCIT
git clone https://github.jpl.nasa.gov/HCIT/go-hcit
```

All external dependencies are bundled in the `vendor` directory, so you will always be able to download go 1.x and compile a program.

If you need to modify a program, cd from `go-hcit` to `/cmd/<the program>` and edit `main.go`.  Then `cd ..` and compile:

```sh
env GOOS=linux GOARCH=amd64 go build main.go
```

The `env` unix command sets environemnt variables for the current command only.  `GOOS`, "go operating system" should be appropriate for the machine you intend to run the software on.  `GOARCH` is the processor architecture, which should generally be 386 (32-bit) or amd64 (64-bit).  The complete list of acceptable values for these constants can be found at https://golang.org/doc/install/source#environment

Note that go supports cross compilation, so compiling for linux or windows from a mac is a nonissue (excluding the andor module).  Also note that as of now (late 2019) operating systems are beginning to drop support for 32-bit binaries, so generally you should use `GOARCH=amd64`.


## Binary servers

see the `cmd` directory for the top-level source code of the server binaries.  When this project reaches maturity,
a collection of the binaries for various architectures and platforms will be stored on the S383 netowrk.  For now, build
a binary to use it.

## Documentation

To view the documentation for the `Go` code, cd to the root of this repository under `$GOPATH`, then run `godoc -http=:6060` and visit http://localhost:6060/pkg/github.jpl.nasa.gov/HCIT/go-hcit/ in your browser.

To view the documentation for HTTP clients, you can build envsrv and visit http://<envsrv-url>/static/docs.html, or use [swagger-ui](https://github.com/swagger-api/swagger-ui) to view and edit the docs locally.

## development status

| Device            | driver | server |
| :---              | :----: |  ---:  |
| JPL DM Controller |        |        |
| BMC commercial    | X      |  X     |
| Andor cameras     | X      |  X     |
| other cameras (?) |        |        |
| Newport EPS       |  X     |  X     |
| Newport XPS       |  X     |        |
| Aerotech Ensemble |  X     |  X     |
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
