/*Package newport provides an HTTP server for ESP and XPS series motion controllers.

This package could use some polish, it feels like there is more code here than needed.

When using LTA actuators that have been treated less than perfectly, you may get
lots of "following errors."  The below text is lifted from the programmer's
manual and may help correct them.

6.2.3 Correcting Following Errors

If the system is stable and the user wants to improve performance, start with
the current PID parameters. The goal is to reduce following error during motion
and to eliminate it at stop.

Guidelines for further tuning (based on performance starting point and desired outcome)
are provided in the following paragraphs.

Following Error Too Large

This is the case of a soft PID loop caused by low values for Kp and Kd. It is
especially common after performing the procedures described in paragraph 6.2.2.
First increase Kp by a factor of 1.5 to 2. Repeat this operation while
monitoring the following error until it starts to exhibit excessive ringing
characteristics (more than 3 cycles after stop). To reduce ringing, add some
damping by increasing the Kd parameter.  Increase it by a factor of 2 while
monitoring the following error. As Kd is increased, overshoot and ringing will
decrease almost to zero.

NOTE
Remember that if acceleration is set too high, overshoot cannot be completely
eliminated with Kd.

If Kd is further increased, at some point oscillation will reappear, usually at
a higher frequency. Avoid this by keeping Kd at a high enough value, but not so
high as to re-introduce oscillation.  Increase Kp successively by approximately
20% until signs of excessive ringing appear again.  Alternately increase Kd and
Kp until Kd cannot eliminate overshoot and ringing at stop. This indicates Kp is
larger than its optional value and should be reduced. At this point, the PID
loop is very tight.  Ultimately, optimal values for Kp and Kd depend on the
stiffness of the loop and how much ringing the application can tolerate.

NOTE
The tighter the loop, the greater the risk of instability and oscillation when
load conditions change.

Errors At Stop (Not In Position)

If you are satisfied with the dynamic response of the PID loop but the stage
does not always stop accurately, modify the integral gain factor Ki. As
described in the Motion Control Tutorial section, the Ki factor of the PID works
to reduce following error to near zero. Unfortunately it can also contribute to
oscillation and overshoot. Change this parameter carefully, and if possible,
in conjunction with Kd.

Start with the integral limit (IL) set to a high value and Ki value at least two
orders of magnitude smaller than Kp. Increase its value by 50% at a time and
monitor overshoot and final position at stop.

If intolerable overshoot develops, increase the Kd factor. Continue increasing
Ki, IL and Kd alternatively until an acceptable loop response is obtained. If
oscillation develops, immediately reduce Ki and IL.  Remember that any finite
value for Ki will eventually reduce the error at stop. It is simply a matter of
how much time is acceptable for the application. In most cases it is preferable
to wait a few extra milliseconds to get to the stop in position rather than have
overshoot or run the risk of oscillations.

Following Error During Motion

This is caused by a Ki, and IL value that is too low. Follow the procedures in
the previous paragraph, keeping in mind that it is desirable to increase the
integral gain factor as little as possible.

6.2.4 Points to Remember

• Use the Windows-based "ESP_tune.exe" utility to change PID parameters and to
visualize the effect. Compare the results and parameters used with the previous
iteration.

• The ESP301 controller uses a servo loop based on the PID with velocity and
acceleration feed-forward algorithm.

• Use the lowest acceleration the application can tolerate. Lower acceleration
generates less overshoot.

• Use the default values provided with the system for all standard motion
devices as a starting point.

• Use the minimum value for Ki, and IL that gives acceptable performance. The
integral gain factor can cause overshoot and oscillations.

*/
package newport
