# andor-http server burst semantics

This document will focus on the Python client to the andor-http servers.  The andor servers have functions that enable capturing bursts ("video").  These use the appropriate features of the cameras to do so, and these frames are captured much faster than.  Andor cameras in particular have on-board memory which they use to buffer frames, 4GB on a Neo (the sCMOS cameras).  The server also allows you to use a buffer, measured in images, of user-defined size which is configured with each burst capture.  We will use a custom version of the server with some print statements embedded in the burst capture to demonstrate.

First, let's look at the Burst function:

```
>>> help(cam.burst)
Help on method burst in module andor:

burst(frames, fps, serverSpool=0) method of andor.Camera instance
    Take a burst of images, returned as a generator of 2D arrays.

    Parameters
    ----------
    `frames` : `int`
        number of frames to take in the sequence
    `fps` : `float`
        framerate to use.  Ensure it is supported by the camera
    serverSpool : `int`
        size of the spool (in frames) to use on the server to buffer,
        if the client can't keep up.  If Zero, the spool size is set to
        frames*fps, which may cause out of memory errors.

    Returns
    -------
    `generator`
        a generator yielding 2D ndarrays.  An exception may be raised while
        iterating it if one is encountered on the server.
```
A [generator](https://wiki.python.org/moin/Generators) in Python is, essentially, a list you can iterate through once that never exists in memory all at once.  Many common functions in python are generators, like `range` (`xrange` in py2).  The generator returned by `burst` pulls the frames from the server as you iterate though them, allowing you to stream sequences of unbounded size.

This means that the image are captured beginning when the request is received by the server, regardless of when you stream them to the client, if the spool(s) can hold them:

```python
>>> cam.burst(100, 50)
<generator object Camera.burst at 0x7f9369144660>

(server log) taking 100 frames took 4.447483937s
```

No frames have been sent to the client, but they are all captured and now reside in the server's spool, off the camera.  A new call to burst without draining the generator will discard the previous captures.

The third (`serverSpool`) parameter is very important and tunes the behavior.  If it is zero, the spool size is set to be the length of the burst.  The server allocates dynamically, so this will not result in out of memory, even for larger than memory sequences, provided you read fast enough to keep up.

The behavior of Andor cameras is they work at the specified framerate until the on-board memory is full, then slow down to capture one new frame as (on-board memory) is available to hold it.  So, if you use a small spool (e.g. unity, which is effectively no spool), the frame rate is determined entirely by the latency of (camera->server->client + processing):


```python
>>> frames = [i for i in cam.burst(100, 50, 1)]

(server log) taking 100 frames took 18.081286173s
```

18 seconds for 100 frames at 11MB per frame works out to about 50-60MB/s, which is likely limited by the fact that we share a 100MB/s ethernet connection on the network.  Because the spool size was one, we did not actually capture at 50fps; the camera buffer filled, then the camera captured at a rate determined by the latency of getting the frames into the client.

If the spool size is increased, the frames are captured faster even if the client only can ingest at 50-60MB/s:
```python
>>> frames = [i for i in cam.burst(100, 50, 25)]

(server log) taking 100 frames took 13.605448131s
```
In this case, 1.1GB of data was generated in 13.6s, ~80MB/s on average.  Using a spool near (or equal to the number of frames), results in the images being captured faster:

```python
>>> frames = [i for i in cam.burst(100, 50, 90)]

(server log) taking 100 frames took 4.448390939s
```
In this case, 10 images can be returned to the client before the spool fills, so frame capture time is minimized.  You can see that the time was consistent to within a millisecond, because the capture time is really tied to the camera's on-board clock.

This matters much more for very large sequences:
```python
>>> frames = [i for i in cam.burst(10000, 50, 1)]

(server log) taking 10000 frames took 33.4min
```
Using `htop`, you can see the server is using a small amount of memory during this time, because the data flow is quite slow.

Note a-priori before this next block that 10,000 frames at 11MB/ea is about 100GB, more or less all of the memory on most of the S383 machines.
```python
>>> frames = [i for i in cam.burst(10000, 50)] # no third parameter, spool size = burst size

(server log) taking 10000 frames took 3.4min
```

In this case, 10,000 frames over 3 and a half minutes is very close to the nominal 50fps.

## exact frame times

There is not presently a way to know exactly what time each frame was taken, so doing things like temporal spectral analysis is compromised by timing uncertainty.  This can be added in the future if desired.
