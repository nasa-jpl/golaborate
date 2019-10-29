# andor

This package uses CGo to work with the SDK provided by andor themselves.  This makes compilation and distribution a bit more difficult than a pure Go program.  To compile on windows, you must follow the rules established by CGo:

1.  Have a `gcc` compiler installed, for example `mingw-w64`
2.  Have `gcc.exe` on the `PATH`.
