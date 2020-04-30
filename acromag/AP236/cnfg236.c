
#include <unistd.h>
#include "../apcommon/apcommon.h"
#include "AP236.h"

/*
{+D}
    SYSTEM:		Library Software

    FILENAME:		cnfg236.c

    MODULE NAME:	cnfg236 - configure AP236 board

    VERSION:		A

    CREATION DATE:	12/01/15

    CODED BY:		FJM

    ABSTRACT:		This module is used to perform the configure function
			for the AP236 board.

    CALLING
	SEQUENCE:	cnfg236(ptr);
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
    for the AP236 board.  A pointer to the Configuration Block will
    be passed to this routine.  The routine will use a pointer
    within the Configuration Block to reference the registers
    on the Board.  Based on flag bits in the Attribute and
    Parameter Flag words in the Configuration Block, the board
    will be configured and various registers will be updated with
    new information which will be transfered from the Configuration
    Block to registers on the Board.
*/



void cnfg236(struct cblk236 *c_blk, int channel)
{

/*
    declare local storage
*/

    uint32_t control;		/* control register */

/*
    ENTRY POINT OF ROUTINE:
    Build up control
*/

    if(c_blk->opts.chan[channel].ParameterMask & 0x80) /* Full Device Reset */
    {
	control = FullResetWrite << 16;	/* initialize control register write value */
	output_long( c_blk->nHandle, (long*)&c_blk->brd_ptr->dac_reg[channel], control );
	usleep((useconds_t) 2);	/* write delay */
    }

    if(c_blk->opts.chan[channel].ParameterMask & 0x40) /* Data Reset */
    {
	control = DataResetWrite << 16;	/* initialize control register write value */
	output_long( c_blk->nHandle, (long*)&c_blk->brd_ptr->dac_reg[channel], control );
	usleep((useconds_t) 2);	/* write delay */
    }


    control = WriteControl << 16;	/* initialize control register write value */

    if(c_blk->opts.chan[channel].ParameterMask & 0x10) /* Clear Voltage */
       control |= (c_blk->opts.chan[channel].ClearVoltage << 9);


    if(c_blk->opts.chan[channel].ParameterMask & 0x08) /* 5% Overrange */
       control |= (c_blk->opts.chan[channel].OverRange << 8);


    if(c_blk->opts.chan[channel].ParameterMask & 0x04) /* Thermal Shutdown */
       control |= (c_blk->opts.chan[channel].ThermalShutdown << 6);


    if(c_blk->opts.chan[channel].ParameterMask & 0x02) /* Power-up Voltage */
       control |= (c_blk->opts.chan[channel].PowerUpVoltage << 3);


    if(c_blk->opts.chan[channel].ParameterMask & 0x01) /* Output Range */
       control |= c_blk->opts.chan[channel].Range;

    output_long( c_blk->nHandle, (long*)&c_blk->brd_ptr->dac_reg[channel], control );
    usleep((useconds_t) 2);	/* write delay */
}

