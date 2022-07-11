# golab

This monorepo contains a number of packages written predominantly in the Go programming language for interacting with lab hardware -- sensors, motion controllers, cameras, test and measurement equipment like digital multimeters and ADC/DAC devices, and so forth.  These packages include HTTP server extensions that allow simple and painless communication with a server driving the hardware.

It also includes some lower level libraries for allowing transparent use of serial or TCP connections to these devices and connection keep-alive behavior on either connection type, as well as graceful backoff in the event of devices rejecting connection thrashing.  These lower level features increase the resilience of the code.

See hardware.csv for a list of supported hardware.

Users likely need not interact with this repository at all, but simply run the appropriate executable acquired from [releases](https://github.com/nasa-jpl/golaborate/releases).

See [golaborate-docs](https://github.com/nasa-jpl/golaborate-docs) for getting started, etc.

## How to Use this

Compile the binaries that interest you from `cmd`, or obtain a binary.  On S383's network, `/home/bdube/golab-releases` contains 64-bit linux binaries.  The code may also be compiled for MacOS and Windows.  To compile, simply download go and type `go build` in `cmd/multiserver` or any of the other cmd folders.  That's it; a static binary will be made with the same name as the folder, which you can run from any computer.  Use `go build GOOS=windows`, or replace `windows` with `darwin` or `linux` to build for a different operating system.  Go will build for the OS you are using by default.

There is a server for Andor SDK2 cameras (iXON EECCD) and SDK3 cameras (Neo, other sCMOS), as well as multiserver, which provides access to all the other devices supported by golab. in one binary.

The servers behave as standard unix binaries, displaying help with no argument and taking several different arguments, e.g.:

```
$ ./andorhttp3
andor-http exposes control of andor Neo cameras over HTTP
This enables a server-client architecture,
and the clients can leverage the excellent HTTP
libraries for any programming language,
instead of custom socket logic.

Usage:
        andor-http <command>

Commands:
        run
        help
        mkconf
        conf
        version
```

Each is configured with a human readable [yaml](https://github.com/darvid/trine/wiki/YAML-Primer) file.

Clients in Python and matlab are available in the [golab-clients](https://github.com/nasa-jpl/golaborate-clients) repository.  The clients are about 1/10th as much code as golab itself.  This is a tremendous labor savings when supporting a new language, maximizing engineer/scientist productivity.

## What is Here

Golab contains 'drivers' for the devices listed in hardare.csv.  These consist of a Go type with the same name as the device, for example `newport.XPS`, created with `newport.NewXPS(...)`.  These types have methods that provide the functionality of the device.  Connection details are internalized and not exposed to the user.  These are about 500 lines of code per device.  These live in the various folders (packages) of this repository.

There is an additional folder, `generichttp,` which defines a generic interface to a given type of device, such as a motion controller, camera, oscilloscope, etc.  The types discussed above implement these interfaces.  An HTTP/JSON wrapper around these types is provided in generichttp.

As is conventional with Go, the `cmd` directory contains the code for binary servers which expose these devices over an
HTTP interface.

## Documentation

To view the documentation for the `Go` code, from the root of the `$GOPATH`, then run `godoc -http=:6060` and visit http://localhost:6060/pkg/github.com/nasa-jpl/golaborate/ in your browser.

You will most likely interact primarily through the various client repositories, which each have their own documentation.  Each `cmd` binary behaves like a standard unix binary with its own `help` command that explains the configuration files, which are [YAML](https://getopentest.org/reference/yaml-primer.html).

## Who's Using Golab

- Roman-CGI
- PIAACMC
- Decadal Survey Testbed
- EMIT
- MISE
- HVM3
