# andor

This package uses CGo to work with the SDK provided by andor themselves.  This makes compilation and distribution a bit more difficult than a pure Go program.  To compile on windows, you must follow the rules established by CGo:

1.  Have a `gcc` compiler installed, for example `mingw-w64`
2.  Have `gcc.exe` on the `PATH`.

# setup of the executable on a new computer

To set up binaries using these packages on a new machine, Andor's drivers must be installed and visible to the executable.  Using Andor's provided should take care of that for you, but if not put the .so files (linux) or .dll files (windows) in the same directory as the executable.
