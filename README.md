# go-hcit
 golang servers/services for S383's high contrast imaging testbeds.  This is set up as a monorepo and contains several packages.  Below is a somewhat infrequently maintained index of the packages and what they enable.  Each type of sensor has a `ReadAndReplyWithJSON` method which implements `http.Handler`

 ### commonpressure

 refactored, common logic for working with pressure sensors.

 ### fluke

 Reading from Fluke 1620a "DewK" temp/humidity sensors over TCP/IP or serial.

 ### granville-phillips

 Reading from GP375 pressure meters over serial.

 ### Lakeshore

 Reading from a 332 sensor/heater controller.

 ### Lesker

 Reading from KJC300 pressure meters.

 ### thermocube

 Reading from Thermocube 200~400 series chillers.

 In /cmd, there is the source for several executables:

 ### envsrv

 This server has routes for each sensor on OMC/GPCT/DST and allows them to be queried via HTTP.

 ### zygo

 This service enables remote measurement with Zygo interferometers via HTTP.
