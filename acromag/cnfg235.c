
#include <unistd.h>
#include "apcommon.h"
#include "AP235.h"

/*
{+D}
    SYSTEM:		Library Software

    FILENAME:		cnfg235.c

    MODULE NAME:	cnfg235 - configure AP235 board

    VERSION:		A

    CREATION DATE:	12/01/15

    CODED BY:		FJM

    ABSTRACT:		This module is used to perform the configure function
			for the AP235 board.

    CALLING
	SEQUENCE:	cnfg235(ptr);
				where:
				ptr (pointer to structure)
				Pointer to the configuration block structure.

    MODULE TYPE:    void

    I/O RESOURCES:

    SYSTEM
	RESOURCES:

    MODULES
	CALLED:

    REVISIONS:

  DATE	  BY	    PURPOSE
-------  ----	------------------------------------------------

{-D}
*/


/*
    MODULES FUNCTIONAL DETAILS:

    This module is used to perform the "configure board" function
    for the AP235 board.  A pointer to the Configuration Block will
    be passed to this routine.  The routine will use a pointer
    within the Configuration Block to reference the registers
    on the Board.  Based on flag bits in the Attribute and
    Parameter Flag words in the Configuration Block, the board
    will be configured and various registers will be updated with
    new information which will be transfered from the Configuration
    Block to registers on the Board.
*/



void cnfg235(struct cblk235 *c_blk, int channel)
{

/*
    declare local storage
*/

    uint32_t control;		/* control register */
    uint32_t temp;

/*
    ENTRY POINT OF ROUTINE:
    Build up control
*/

    /* make sure interrupts for this channel are disabled */
    output_long(c_blk->nHandle, (long*)&c_blk->brd_ptr->AXI_ClearInterruptEnableRegister, (long)( 1 << channel));

    control = FullResetWrite << 16;	/* initialize control register write value */
    output_long( c_blk->nHandle, (long*)&c_blk->brd_ptr->DAC[channel].DirectAccess, control );
    usleep((useconds_t) 2);	/* write delay */

    control = DataResetWrite << 16;	/* initialize control register write value */
    output_long( c_blk->nHandle, (long*)&c_blk->brd_ptr->DAC[channel].DirectAccess, control );
    usleep((useconds_t) 2);	/* write delay */

    control = WriteControl << 16;	/* initialize channel control register write value */

    control |= (c_blk->opts.chan[channel].ClearVoltage << 9); /* Clear Voltage */

    control |= (c_blk->opts.chan[channel].OverRange << 8); /* 5% Overrange */

    control |= (c_blk->opts.chan[channel].ThermalShutdown << 6); /* Thermal Shutdown */

    control |= (c_blk->opts.chan[channel].PowerUpVoltage << 3); /* Power-up Voltage */

    control |= c_blk->opts.chan[channel].Range; /* Output Range */

    output_long( c_blk->nHandle, (long*)&c_blk->brd_ptr->DAC[channel].DirectAccess, control );

    /* Underflow Clear in DAC channel status */
    output_long( c_blk->nHandle, (long*)&c_blk->brd_ptr->DAC[channel].Status, c_blk->opts.chan[channel].UnderflowClear << 3 );

    /* timer divider value */
    output_long( c_blk->nHandle, (long*)&c_blk->brd_ptr->TimerDivider, c_blk->TimerDivider );

    /* Trigger Direction */
    temp = input_long(c_blk->nHandle, (long *)&c_blk->brd_ptr->CommonControl);
    temp &= 0xFFFFFFF7;		/* clear trigger direction */
    temp |= c_blk->TriggerDirection << 3;
    output_long( c_blk->nHandle, (long *)&c_blk->brd_ptr->CommonControl, (long)temp );

    /* configure channel X control register */
    control = c_blk->opts.chan[channel].OpMode;	/* get Operating Mode */

    /* DAC_FIFO_DMA is an abstraction that differentiates DMA transfers from CPU transfers */
    /* only the DAC_FIFO mode exists, the DAC FIFO can be written by CPU or DMA */
    if(control == DAC_FIFO_DMA)
       control = DAC_FIFO;

    control |= (c_blk->opts.chan[channel].TriggerSource << 2); /* Trigger Source */
    output_long( c_blk->nHandle, (long *)&c_blk->brd_ptr->DAC[channel].Control, control );

    switch(c_blk->opts.chan[channel].OpMode)
    {
       case DAC_SB:	    /* these modes can be interrupt driven */
       case DAC_FIFO:
       case DAC_FIFO_DMA:
            /* Interrupt Source enable/disable DAC_FIFO, DAC_FIFO_DMA, or Single Burst interrupt */
            if( c_blk->opts.chan[channel].InterruptSource == FIFO_SBURST )
                output_long(c_blk->nHandle, (long*)&c_blk->brd_ptr->AXI_SetInterruptEnableRegister, (long)( 1 << channel));
       break;
    }
}

