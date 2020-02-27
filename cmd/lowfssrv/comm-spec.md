# LOWFS server

This document describes communication between the LOWFS server and the greater world, as well as the hardware and software architecture.  The design goals of this server are to satisfy the demonstration requirements:
- (close the loop at 500Hz)
- play back disturbances on the jitter mirror

While enabling some new capabilities:
- enable high flexibility into Zernike (or other mode) estimation (Reconstruction)
- enable easy maintanance of the system

Point of Contact [Brandon Dube (383D)](mailto:brandon.dube@jpl.nasa.gov)


## Communication

The LOWFS server is extremely multi-threaded.  While the languge it is built in (Go) does not use the ideas of threads, we will use threads interchangeably with Goroutines (which the Go runtime multiplexes onto a pool of threads it manages).  The following actions are [concurrent](https://en.wikipedia.org/wiki/Concurrent_computing)

- HTTP/JSON server providing access to the camera over the standard andorhttp interface
- HTTP server providing control over meta parameters of the control system,
- - loop period (default: 2 ms)
- - upload CSV file with disturbances to play back
- - start/stop/pause playback of disturbances
- [ØMQ](https://zeromq.org/) server providing low latency (<10μs) access to camera data and feed back to control system
- operation of inner control loop based on strain gauge data

The HTTP servers provide the usual documentation expected of a go-hcit HTTP server while they are running.  The camera can be accessed as usual via the [client-andor](https://github.jpl.nasa.gov/HCIT/client-andor) library, while an additional client is provided that utilizes the ØMQ communication layer for low latency access to camera buffers.  This client controls the loop parameters as well as jitter mirror playback in addition to providing the camera frames.

If you wish to create your own client to the ØMQ server, you need only the following guide for communication:

- the HTTP server runs on port 8000
- the ØMQ server runs on port 8001
- the ØMQ server replies to the query, "frame?" with no terminators or other extras (e.g. NULL termination in C), and replies with the buffer from the camera, which you may expect to be WxHx2 bytes in size.  For a 60x60 image, this is 7.2kB.  The buffer has already been stripped of padding, and represents little endian uint16s.  ZMQ deals with transport and framing for you.  Use the HTTP API to the camera to learn what the width and height of the AOI are to reshape it as appropriate.
- the query "fsm:i,j,k", where i,j,k are ASCII encoded floating point numbers.  Any format may be used.  The server will send these commands to the FSM.  The reply will be the byte 6 (acknowledge), or the content of an error message as UTF-8 encoded string.  This communication channel is part of the phase locked loop. and is not the expected path to be used to send single commands to the FSM or JM.  Use the HTTP for that.

If you do not wish to roll your own, then the python API should be used as thus:


```python
from lowfsclient import lowfs

# set up communication to the server
lowfs.connect()
# later
# lowfs.disconnect()

sentinel = True
while sentinel:
    data = lowfs.frame() # data is a 60x60 uint16 numpy array
    zernikes = your_reconstruction_pipeline() # say, 1x13 vector of f64

    volts = your_calibration() # should produce a length 3 iterable
    lowfs.fsm_feedback(volts)

# to disturb the system
lowfs.upload_jm_profile(local_path_to_csv) # volts

# the JM will begin playing back the disturbance,
# looping back to the beginning when it is over
lowfs.jm_start()

# the JM will stop where it is,
# but not reset the "cursor" in the disturbance
lowfs.jm_pause()

# the JM will stop where it is, but not "recenter"
lowfs.jm_stop()

# to send singular commands
# in volts
lowfs.jm_command((1,2,3)) # 1,2,3 is any length 3 iterable of volts

lowfs.fsm_command([1,2,3]) # ditto, see that tuple or list or array does not matter
```

Note that while TCP sockets are used for communication, running reconstruction on a computer other than the one running the server will probably introduce enough latency into the system to impede performance to be below 500Hz.  The server will operate below 500Hz without error or complaint; the phase lock on the loop only delays to ensure the maximum rep rate is 500Hz (or as specified).

ØMQ provides spectacularly low latency on the same node (< 10 _microseconds_) but the physical transport layer is not as quick as that between multiple machines (> 250 μs).  If your reconstruction is fast enough to close the loop with the additional latency, the system will still work at 500Hz.
