/*Package emccd extends the andor package to provide a safe operational mode
of EMCCD cameras.  The following behavior is implemented:

The camera is initialized with EM gain of 1x, and the gain is walked up the
TargetGain setting over the course of several minutes with exposures taken
and evaluated for signal level in between.  If the detector approaches
saturation, a scenario that will lead to accelerated ageing, the gain is clamped
and an error returned.  If no error is found and the gain was never clamped,
the error returned is nil and the gain truly reached the requested level.
*/
package emccd
