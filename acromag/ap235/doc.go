/*Package ap235 provides an interface to Acromag AP235 16-bit waveform DAC modules

Some performance-limiting design changes are made from the C SDK provided
by acromag.  Namely: there are Go functions for each channel configuration,
and a call to any of them issues a transfer of the configuration to the board.
This will prevent the last word in performance under obscure conditions (e.g.
updating the config 10,000s of times per second) but is not generally harmful
and simplifies interfacing to the device, as there can be no "I updated the
config but forgot to write to the device" errors.

Basic usage is as followed:
 dac, err := ap235.New(0) // 0 is the 0th card in the system, in range 0~4
 if err != nil {
 	log.fatal(err)
 }
 defer dac.Close()
 // single channel, immediate mode (output immediately on write)
 // see method docs on error values, this example ignores them
 // there is a get for each set
 ch := 1
 dac.SetRange(ch, TenVSymm)
 dac.SetPowerUpVoltage(ch, ap235.MidScale)
 dac.SetClearVoltage(ch, ap235.MidScale)
 dac.SetOverTempBehavior(ch, true) // shut down if over temp
 dac.SetOverRange(ch, false) // over range not allowed
 dac.SetOutputSimultaneous(ch, false) // this is what puts it in immediate mode
 dac.Output(ch, 1.) // 1 volt as close as can be quantized.  Calibrated.
 dac.OutputDN(ch, 2000) // 2000 DN, uncalibrated

 // multi-channel, synchronized
 chs := []int{1,2,3}
 // setup
 for _, ch := range chs {
 	dac.SetRange(ch, TenVSymm)
	dac.SetPowerUpVoltage(ch, ap235.MidScale)
	dac.SetClearVoltage(ch, ap235.MidScale)
	dac.SetOverTempBehavior(ch, true)
	dac.SetOverRange(ch, false)
 	dac.SetOutputSimultaneous(ch, true)
 }
 // in your code
 dac.OutputMulit(chs, []float64{1, 2, 3} // calibrated
 dac.OutputMultiDN(chs, []uint16{1000, 2000, 3000}) // uncalibrated

*/
package ap235

/* steaming workflow, from AP235 man:
1. Start in a known state by writing the Control Register with the
	Software Reset bit set to logic ‘1’.
2. Reset the DACs by writing the Control Register with the DAC reset bit set
	to logic ‘1’.
3. Configure each DAC channel by writing to the appropriate
	Channel Direct Access register. Set the output range, power-up voltage,
	clear voltage, and data format.
4. Set the initial output voltage for all DACs by writing the
	Control Register with the DAC clear bit set to logic ‘1’. This will
	power up the DAC outputs. The DACs will output the voltage configured
	with the previous step.
5. Set the FIFO size for each channel by writing the Channel Start Address
	and Channel End Address registers. If all channels will be outputting
	data at the same frequency, make all the FIFOs equal size.
6. Configure the operating mode of each channel as FIFO mode, and set the
	appropriate bits to select the trigger source.
7. Initialize the DMA scatter-gather descriptor chain list in block RAM. Up
	to sixteen descriptors could be needed each time a transfer is initiated.
	All the host memory addresses written to the descriptors must take into
	consideration the address translation that is performed by the PCIe
	interface. The Next Descriptor Pointer field must be set for each of the
	sixteen descriptors. The destination address will be the appropriate
	Channel FIFO register. Set the bytes to transfer field in the descriptor
	to one half the size of the sample memory allocated to each channel.
	The source address will be a host memory address where the next set of
	sample data for each channel is stored. Write zeroes to the
	Transfer Descriptor Status Word for each descriptor to indicate that it
	has not completed.
8. Reset the CDMA by writing the Reset bit in the CDMA Control Register.
9. Poll the CDMA control register until the Reset bit indicates reset not in
	progress.
10. Configure the CDMA by writing the CMDA Control Register with
	Tail Pointer Mode enabled, Scatter Gather Mode selected, Key Hole write
	enabled, and Cyclic BD Disabled.
11. Write the address of the first scatter-gather descriptor to the CDMA
	Current Descriptor Pointer Register.
12. Write the address of the descriptor set up for channel fifteen to the
	CDMA Tail Descriptor Pointer Register. This initiates the DMA transfers.
13. Poll the CDMA Status Register until the CDMA idle bit indicates the CMDA
	is in the idle state. Each of the FIFOs are now half full.
14. Write the Interrupt Enable register with 0xFFFF. This enables each DAC
	channel to generate interrupts. Since each channel is configured in
		FIFO mode, an interrupt will be generated when any of the channels’
		FIFOs becomes half full. Also, note the CDMA interrupt is not enabled.
15. Write the following fields of the Master Enable Register:
	Master IRQ Enable = 1
	Hardware Interrupt Enable = 1
16. Write the Waveform Output Enable bit in the Control Register to start
	waveform output. The DACs will output the data stored in their FIFOs at
	the rate of the trigger pulses.
17. Wait for an interrupt from the AP2x5 module.
18. Read the Interrupt Pending Register. Store the value read for later use
	in the DMA complete interrupt handler. For each DAC channel interrupt
	bit in the Interrupt Pending Register that is set to a logic ‘1’ set up
	the scatter-gather descriptor to transfer up to one half of the
	channel’s FIFO size.
19. For each DAC channel interrupt bit in the Interrupt Pending Register
	that is not set to a logic ‘1’, remove the channel’s descriptor from the
	scatter-gather chain.
20. Write the address of the scatter-gather descriptor of the first channel
	requiring service to the CDMA Current Descriptor Pointer Register.
21. Write the following fields of the CDMA control register:
	Scatter Gather Mode = 1
	Key Hole Write = 1
	Cyclic BD Enable = 0
	Interrupt on Complete Interrupt Enable = 1
	Interrupt on Delay Timer = 0
	Interrupt on Error Interrupt Enable = 1
	Interrupt Threshold Value = number of descriptors to transfer
	Interrupt Delay Timeout = 0
22. Write 0x10000 to the Interrupt Enable Register. This disables all DAC
	channel interrupts and enables the CDMA interrupt.
23. Write the address of the descriptor of the last channel requiring
service to the CMDA Tail Descriptor Pointer Register. This will initiate the
DMA transfers.
24. Wait for an interrupt from the AP2x5 module.
25. Read the CMDA status register. Check for error bits that are set.
26. Write the Interrupt Acknowledge Register with the saved value from the
	Interrupt Pending Register from above. This will clear the interrupts
	for the channels that were serviced by the DMA transfer.
27. Write 0xFFFF to the Interrupt Enable Register to re-enable the DAC
	interrupts.
*/
