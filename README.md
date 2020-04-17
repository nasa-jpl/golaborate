# golab

This monorepo contains a number of packages written predominantly in the Go programming language for interacting with lab hardware -- sensors, motion controllers, cameras, and deformable mirrors.  These packages include HTTP server extensions that allow simple and painless communication with a server driving the hardware.  The Andor, BMC, and Thorlabs ITC code uses CGo and is less portable for that reason.

It also includes some lower level libraries for allowing transparent use of serial or TCP connections to these devices and connection keep-alive behavior on either connection type, as well as graceful backoff in the event of devices rejecting connection thrashing.

See hardware.csv for a list of supported hardware.
## Installation and Compilation

Most of these servers are written in Go.  Go has a statement of absolute compatability within the 1.x version series, so you may use any version that is recent enough for the dependencies.  At the time of writing, this is Go 1.13.

To install golang, grab the binaries from http://golang.org/dl, use your system package manager, or [brew](https://brew.sh/) on MacOS.  For brew:

```
brew install go  # you can use go@1.9 to get v1.9
git clone https://github.jpl.nasa.gov/bdube/golab
```

If you need to modify a program, cd from `golab` to `/cmd/<the program>` and edit `main.go`.  Then `cd ..` and compile:

```sh
env GOOS=linux GOARCH=amd64 go build main.go
```

The `env` unix command sets environment variables for the current command only.  `GOOS`, "go operating system" should be appropriate for the machine you intend to run the software on.  `GOARCH` is the processor architecture, which should generally be 386 (32-bit) or amd64 (64-bit).  The complete list of acceptable values for these constants can be found at https://golang.org/doc/install/source#environment.

Note that go supports cross compilation, so compiling for linux or windows from a mac is a nonissue (excluding the andor module).  Note that as of now (late 2019) operating systems are beginning to drop support for 32-bit binaries, so generally you should use `GOARCH=amd64`.


## Binary servers

see the `cmd` directory for the top-level source code of the server binaries.  When this project reaches maturity,
a collection of the binaries for various architectures and platforms will be stored on the S383 netowrk.  For now, build
a binary to use it.

## Documentation

To view the documentation for the `Go` code, cd to the root of this repository under `$GOPATH`, then run `godoc -http=:6060` and visit http://localhost:6060/pkg/github.jpl.nasa.gov/bdube/golab/ in your browser.

You will most likely interact primarily through the various client repositories, which each have their own documentation.  Each `cmd` binary behaves like a standard unix binary with its own `help` command that explains the configuration files, which are [YAML](https://getopentest.org/reference/yaml-primer.html).
