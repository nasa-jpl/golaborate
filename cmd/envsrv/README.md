# envsrv

This source code defines the application layer for a server which makes communication with a wide range of devices simpler and language agnostic.  For more information on the types of devices that are supported, see the main `go-hcit` documentation.  The envsrv binary is highly concurrent and supports far more concurrent users (and devices) than its intended audience could ever throw at it, even when running on a modest computer.  Lack of race conditions at the level of "working code" is guaranteed, but this does not prevent multiple users from interfering with each others' usage.  This binary serves only to make communicating with these devices easier, it does not perform automatic polling of sensors or any other functions.

# acquiring the server

Before you may use this server, you must have either a copy of the binary that is compiled for your system's operating system and architecture.  Binaries are provided for the following combinations:

- linux:
- - amd64 (64-bit)
- - 386   (32-bit)
- - arm   (ARM processors, like in a raspberry pi)
- darwin (MacOS):
- - amd64
- - 386
windows:
- - arm64
- - 386

But you may build your own binary after installing go-hcit and its dependencies, as described in the go-hcit main documentation.  support for Andor cameras requires `CGo` and having installed the appropriate SDK version on both the build and deployment machines.

# usage

To use the server, make a config file that looks similar to this snippet:

```yaml
Flukes:
  - addr: 192.168.100.71
    url: /zygo-bench

NKTs:
  - addr: 192.168.100.40:2106
    url: /omc/nkt

IXLLightwaves:
  - addr: 192.168.100.40:2106
    url: /omc/ixl-diode

Leskers:
  - addr: 192.168.100.187:2113
    url: /dst/lesker

GPConvectrons:
  - addr: 192.168.100.41:2106
    url: /dst/convectron

# ... etc
```

Then run the server with the path to the config file as an argument:

```sh
./envsrv cfg.yml
```

If the server is deployed adjacent to a `static` folder containing the docs distribution for it, you can visit `http://<server-ip>:<server-port>/static/docs.html` to view the HTTP documentation.
