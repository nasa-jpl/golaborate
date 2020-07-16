# Notes writing (C)Go wrapper over AP235

- AP235 is quite a bit more work to program than AP236.  Waveform memory has created a lot of complexity in their C code that does not exist in the AP236 C code.

- `long` is really a memory address when acromag uses it; int of >= 32 bits.

- AP235 - `scatter_info` is a 4 element tuple of:
- - a pointer (user memory)
- - a data size / length
- - a pointer (PCIe device memory)
- - device number(?)
