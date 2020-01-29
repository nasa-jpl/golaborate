# Installation and Compilation

The BMC driver / C library has fairly compilcated compilation.  It links to a
number of libraries that are bundled, those are taken care of with the CGO flags
in the preamble of bmc.go.  However, CBlas is also needed.

To install it on debian/ubuntu:

```sh
sudo apt install libatlas-base-dev
```

Then you should be able to compile the program, and you shouldn't get panics from
missing libraries when you run it.

`libatlas-base-dev` most likely needs to be installed on each machine you want to
run the BMC server on.
